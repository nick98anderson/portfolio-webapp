// Full-screen pass: one oversized triangle, UVs from clip space.
export const VERT = `
attribute vec2 p;
varying vec2 vUv;
void main() {
	vUv = p * 0.5 + 0.5;
	gl_Position = vec4(p, 0.0, 1.0);
}`;

// Domain-warped fbm drags the picture around like liquid, with a finer
// ripple on top, then soft focus, tint toward --ink and a vignette.
export const FRAG = `
precision highp float;
varying vec2 vUv;
uniform sampler2D uTex;
uniform float uTime, uZoom, uAmp, uSize, uSpeed, uRipple, uTint, uVignette;
uniform vec2 uCover;   // scale that makes the video "object-fit: cover"
uniform vec2 uBlur;    // blur radius in texture UV units (0 = off)
uniform vec3 uInk;

// Smooth value noise + fbm.
float hash(vec2 p) { return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453); }
float noise(vec2 p) {
	vec2 i = floor(p), f = fract(p);
	vec2 u = f * f * (3.0 - 2.0 * f);
	return mix(mix(hash(i), hash(i + vec2(1, 0)), u.x),
	           mix(hash(i + vec2(0, 1)), hash(i + vec2(1, 1)), u.x), u.y);
}
float fbm(vec2 p) {
	float v = 0.0, a = 0.5;
	for (int i = 0; i < 4; i++) { v += a * noise(p); p = p * 2.03 + 11.7; a *= 0.5; }
	return v;
}

// Soft focus: 16 taps on a spiral disk, rotated per pixel so large radii
// turn into fine noise (hidden by the grain) instead of visible rings.
vec3 sampleBlurred(vec2 uv) {
	if (uBlur.x <= 0.0) return texture2D(uTex, clamp(uv, 0.001, 0.999)).rgb;
	float rot = hash(gl_FragCoord.xy) * 6.2831;
	vec3 acc = vec3(0.0);
	for (int i = 0; i < 16; i++) {
		float fi = float(i);
		float r = sqrt((fi + 0.5) / 16.0);
		float a = fi * 2.39996 + rot;
		acc += texture2D(uTex, clamp(uv + vec2(cos(a), sin(a)) * r * uBlur, 0.001, 0.999)).rgb;
	}
	return acc / 16.0;
}

void main() {
	vec2 uv = (vUv - 0.5) * uCover / uZoom + 0.5;
	float t = uTime * uSpeed;

	// Big slow warp: domain-warped fbm gives flowing, melting distortion.
	vec2 q = uv * uSize;
	vec2 w = vec2(fbm(q + vec2(0.0, t)), fbm(q + vec2(5.2, -t * 0.8)));
	vec2 off = vec2(fbm(q + 2.0 * w + vec2(1.7, 9.2) + t),
	                fbm(q + 2.0 * w + vec2(8.3, 2.8) - t)) - 0.5;
	uv += off * uAmp;

	// Fine ripple on top.
	uv += uRipple * vec2(sin(uv.y * 40.0 + uTime * 1.3), cos(uv.x * 34.0 - uTime * 1.1));

	vec3 col = sampleBlurred(uv);
	col = mix(col, uInk, uTint);

	// Vignette: darken toward the edges, strongest in the corners.
	float edge = smoothstep(0.35, 1.0, length(vUv - 0.5) * 1.4142);
	col = mix(col, uInk * 0.45, edge * uVignette);
	gl_FragColor = vec4(col, 1.0);
}`;

import Hero from '../components/Hero';
import VideoWarp from '../components/VideoWarp';
import { asset } from '../lib/asset';

export default function Home() {
	return (
		<>
			<VideoWarp
				src={asset('video/bg.mp4')}
				poster={asset('video/bg-poster.jpg')}
				portraitSrc={asset('video/bg-portrait.mp4')}
				portraitPoster={asset('video/bg-portrait-poster.jpg')}
				controls={import.meta.env.DEV}
			/>
			<Hero />
		</>
	);
}

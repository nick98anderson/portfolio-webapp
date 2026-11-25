import Hero from '../components/Hero';
import VideoWarp from '../components/VideoWarp';
import { useBackground } from '../lib/background';

export default function Home() {
	const bg = useBackground();
	return (
		<>
			{bg && (
				<VideoWarp
					src={bg.src}
					poster={bg.poster}
					portraitSrc={bg.portraitSrc}
					portraitPoster={bg.portraitPoster}
					controls={import.meta.env.DEV}
				/>
			)}
			<Hero />
		</>
	);
}

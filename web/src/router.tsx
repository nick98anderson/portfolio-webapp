import { createBrowserRouter, Navigate, Outlet, ScrollRestoration } from 'react-router';
import Grain from './components/Grain';
import Home from './pages/Home';
import Resume from './pages/Resume';

function Root() {
	return (
		<>
			<Grain />
			<Outlet />
			<ScrollRestoration />
		</>
	);
}

export const router = createBrowserRouter([
	{
		element: <Root />,
		children: [
			{ index: true, element: <Home /> },
			{ path: 'resume', element: <Resume /> },
			// About was folded into the resume page; keep old links working.
			{ path: 'about', element: <Navigate to="/resume" replace /> },
			{ path: '*', element: <Navigate to="/" replace /> },
		],
	},
]);

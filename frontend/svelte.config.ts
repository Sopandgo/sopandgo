import adapter from '@sveltejs/adapter-node';
import type { Config } from '@sveltejs/kit';

const config: Config = {
	kit: {
		// adapter-node will create a 'build' folder when you run 'npm run build'
		adapter: adapter()
	}
};

export default config;

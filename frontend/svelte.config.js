import adapter from '@sveltejs/adapter-node'; // Changed from adapter-auto

/** @type {import('@sveltejs/kit').Config} */
const config = {
    kit: {
        // adapter-node will create a 'build' folder when you run 'npm run build'
        adapter: adapter()
    }
};

export default config;
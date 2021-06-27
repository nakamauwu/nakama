const proxy = require("http2-proxy")

/** @type {import("snowpack").SnowpackUserConfig } */
module.exports = {
	packageOptions: {
		knownEntrypoints: [
			"emoji-picker-element/picker",
			"emoji-picker-element/database",
		],
	},
	plugins: [
		["@snowpack/plugin-optimize", { preloadModules: true }],
	],
	buildOptions: {
		out: "../static/",
		metaUrlPath: "snowpack",
	},
	devOptions: {
		port: 5000,
		open: "none",
	},
	routes: [
		{
			src: "/api/.*",
			dest: (req, res) => {
				let result = proxy.web(req, res, {
					hostname: 'localhost',
					port: 3000,
				})
				if (result instanceof Promise) {
					result = result.catch(() => {
						console.error("server down")
					})
				}
				return result
			},
		},
		{
			match: "routes",
			src: ".*",
			dest: "/index.html",
		},
	],
}

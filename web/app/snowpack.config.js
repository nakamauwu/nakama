const proxy = require("http2-proxy")

/** @type {import("snowpack").SnowpackUserConfig } */
module.exports = {
	buildOptions: {
		out: "../static/",
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

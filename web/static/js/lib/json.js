/**
 * @param {string} text
 * @param {function(any,any):any=} reviver
 */
export function parseJSON(text, reviver) {
	text = String(text).replace(/([^\"]+\"\:\s*)(\d{16,})(\,\s*\"[^\"]+|}$)/g, '$1"$2n"$3')
	return JSON.parse(text, (k, v) => {
		if (typeof v === 'string' && /^\d{16,}n$/.test(v)) {
			v = BigInt(v.slice(0, -1))
		}
		return typeof reviver === 'function' ? reviver(k, v) : v
	})
}

/**
 * @param {any} value
 * @param {(function(any,any):any)|((number|string)[])=} replacer
 * @param {string|number=} space
 */
export function stringifyJSON(value, replacer, space) {
	return JSON.stringify(value, (k, v) => {
		if (typeof v === 'bigint') {
			v = v.toString() + 'n'
		}
		return typeof replacer === 'function' ? replacer(k, v) : v
	}, space).replace(/([^\"]+\"\:\s*)(?:\")(\d{16,})(?:n\")(\,\s*\"[^\"]+|}$)/g, '$1$2$3')
}

export default {
	parse: parseJSON,
	stringify: stringifyJSON,
}

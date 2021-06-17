import { useEffect, useState } from "haunted";

/**
 * @param {function():Promise} fetcher
 * @param {Array} deps
 */
export function useFetch(fetcher, deps, defaultValue = undefined) {
    const [data, setData] = useState(defaultValue)
    const [isFetching, setIsFetching] = useState(true)
    const [err, setErr] = useState(null)

    useEffect(() => {
        setIsFetching(true)
        fetcher().then(setData, setErr).finally(() => {
            setIsFetching(false)
        })
    }, deps)

    return {
        data,
        mutate: setData,
        isFetching,
        err,
    }
}

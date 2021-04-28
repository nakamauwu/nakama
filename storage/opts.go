package storage

// StoreOpts to set while storing.
type StoreOpts struct {
	ContentType     string
	ContentEncoding string
	CacheControl    string
}

// StoreOpt type.
type StoreOpt func(*StoreOpts)

// StoreWithContentType option.
func StoreWithContentType(s string) StoreOpt {
	return func(opts *StoreOpts) {
		opts.ContentType = s
	}
}

// StoreWithContentEncoding option.
func StoreWithContentEncoding(s string) StoreOpt {
	return func(opts *StoreOpts) {
		opts.ContentEncoding = s
	}
}

// StoreWithCacheControl option.
func StoreWithCacheControl(s string) StoreOpt {
	return func(opts *StoreOpts) {
		opts.CacheControl = s
	}
}

package units

const (
	_        = iota             // Ignore the first value (0)
	Kibibyte = 1 << (10 * iota) // 1 KiB = 1024 bytes
	Mebibyte = 1 << (10 * iota) // 1 MiB = 1024 KiB
	Gibibyte = 1 << (10 * iota) // 1 GiB = 1024 MiB
	Tebibyte = 1 << (10 * iota) // 1 TiB = 1024 GiB
	Pebibyte = 1 << (10 * iota) // 1 PiB = 1024 TiB
)

const (
	DiscordLimit = 8 * Mebibyte // 8 MiB
)

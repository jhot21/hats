package passgen

type Passgen struct {
	Version        int
	Passphrase     string
	Salt           string
	Length         int
	NoUppers       bool
	NoNumbers      bool
	NoSpecials     bool
	CustomSpecials string
}

type PassgenOption func(*Passgen)

func WithVersion(version int) PassgenOption {
	return func(p *Passgen) {
		p.Version = version
	}
}

func WithPassphrase(passphrase string) PassgenOption {
	return func(p *Passgen) {
		p.Passphrase = passphrase
	}
}

func WithSalt(salt string) PassgenOption {
	return func(p *Passgen) {
		p.Salt = salt
	}
}

func WithLength(length int) PassgenOption {
	return func(p *Passgen) {
		p.Length = length
	}
}

func WithoutSpecials() PassgenOption {
	return func(p *Passgen) {
		p.NoSpecials = true
	}
}

func WithoutNumbers() PassgenOption {
	return func(p *Passgen) {
		p.NoNumbers = true
	}
}

func WithoutUppers() PassgenOption {
	return func(p *Passgen) {
		p.NoUppers = true
	}
}

func WithCustomSpecials(specials string) PassgenOption {
	return func(p *Passgen) {
		p.CustomSpecials = specials
	}
}

func NewPasswordGenerator(opts ...PassgenOption) *Passgen {
	p := &Passgen{
		Version:        2,
		Length:         40,
		NoUppers:       false,
		NoNumbers:      false,
		NoSpecials:     false,
		CustomSpecials: "!@#$%^&*",
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

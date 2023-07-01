package beepberry

import (
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

const (
	write = 0x80

	power = 0x20
	red   = 0x21
	green = 0x22
	blue  = 0x23
)

type LED struct {
	bus  i2c.BusCloser
	chip *i2c.Dev
}

func NewLED() (*LED, error) {
	if _, err := host.Init(); err != nil {
		return nil, err
	}

	i, err := i2creg.Open("1")
	if err != nil {
		return nil, err
	}

	return &LED{
		bus:  i,
		chip: &i2c.Dev{Addr: 0x1F, Bus: i},
	}, nil
}

func (l *LED) Close() error {
	return l.bus.Close()
}

func (l *LED) On() error {
	_, err := l.chip.Write([]byte{power + write, 0x01})
	return err
}

func (l *LED) Off() error {
	_, err := l.chip.Write([]byte{power + write, 0x00})
	return err
}

func (l *LED) SetColor(r, g, b uint16) error {
	if _, err := l.chip.Write([]byte{red + write, byte(r)}); err != nil {
		return err
	}

	if _, err := l.chip.Write([]byte{green + write, byte(g)}); err != nil {
		return err
	}

	if _, err := l.chip.Write([]byte{blue + write, byte(b)}); err != nil {
		return err
	}

	return nil
}

func (l *LED) IsOn() (bool, error) {
	p := make([]byte, 1)
	if err := l.chip.Tx([]byte{power}, p); err != nil {
		return false, err
	}

	return p[0] > 0, nil
}

func (l *LED) Color() ([]byte, error) {
	r := make([]byte, 1)
	if err := l.chip.Tx([]byte{red}, r); err != nil {
		return nil, err
	}
	g := make([]byte, 1)
	if err := l.chip.Tx([]byte{green}, g); err != nil {
		return nil, err
	}
	b := make([]byte, 1)
	if err := l.chip.Tx([]byte{blue}, b); err != nil {
		return nil, err
	}

	return []byte{r[0], g[0], b[0]}, nil
}

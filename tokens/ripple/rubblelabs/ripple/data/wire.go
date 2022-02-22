package data

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

func (v *Value) Unmarshal(r Reader) error {
	var u uint64
	if err := binary.Read(r, binary.BigEndian, &u); err != nil {
		return err
	}
	v.native = (u >> 63) == 0
	v.negative = (u>>62)&1 == 0
	if v.IsNative() {
		v.num = u & ((1 << 62) - 1)
		v.offset = 0
	} else {
		v.num = u & ((1 << 54) - 1)
		v.offset = int64((u>>54)&((1<<8)-1)) - 97
	}
	return nil
}

func (v *Value) Marshal(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, v.Bytes())
}

func (a *Amount) Unmarshal(r Reader) error {
	a.Value = new(Value)
	if err := a.Value.Unmarshal(r); err != nil {
		return err
	}
	if a.Value.IsNative() {
		return nil
	}
	if err := unmarshalSlice(a.Currency[:], r, "Currency"); err != nil {
		return err
	}
	if err := unmarshalSlice(a.Issuer[:], r, "Issuer"); err != nil {
		return err
	}
	return nil
}

func (a *Amount) Marshal(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, a.Bytes())
}

func (c *Currency) Unmarshal(r Reader) error {
	return unmarshalSlice(c[:], r, "Currency")
}

func (c *Currency) Marshal(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, c.Bytes())
}

func (h *Hash128) Unmarshal(r Reader) error {
	return unmarshalSlice(h[:], r, "Hash128")
}

func (h *Hash128) Marshal(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, h.Bytes())
}

func (h *Hash160) Unmarshal(r Reader) error {
	return unmarshalSlice(h[:], r, "Hash160")
}

func (h *Hash160) Marshal(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, h.Bytes())
}

func (h *Hash256) Unmarshal(r Reader) error {
	return unmarshalSlice(h[:], r, "Hash256")
}

func (h *Hash256) Marshal(w io.Writer) error {
	return binary.Write(w, binary.BigEndian, h.Bytes())
}

func (v *Vector256) Unmarshal(r Reader) error {
	length, err := readVariableLength(r)
	if err != nil {
		return err
	}
	count := length / 32
	*v = make(Vector256, count)
	for i := 0; i < count; i++ {
		if err := (*v)[i].Unmarshal(r); err != nil {
			return err
		}
	}
	return nil
}

func (v *Vector256) Marshal(w io.Writer) error {
	var b []byte
	for _, h := range *v {
		b = append(b, h[:]...)
	}
	return writeVariableLength(w, b)
}

func (v *VariableLength) Unmarshal(r Reader) error {
	length, err := readVariableLength(r)
	if err != nil {
		return err
	}
	*v = make(VariableLength, length)
	return unmarshalSlice(*v, r, "VariableLength")
}

func (v *VariableLength) Marshal(w io.Writer) error {
	return writeVariableLength(w, v.Bytes())
}

func readExpectedLength(r Reader, dest []byte, prefix string) error {
	length, err := readVariableLength(r)
	switch {
	case err != nil:
		return fmt.Errorf("%s: %s", prefix, err.Error())
	case length == 0:
		return nil
	case length == len(dest):
		return unmarshalSlice(dest, r, prefix)
	default:
		return fmt.Errorf("%s: wrong length %d expected: %d", prefix, length, len(dest))
	}
}

func (a *Account) Unmarshal(r Reader) error {
	return readExpectedLength(r, a[:], "Account")
}

func (a *Account) Marshal(w io.Writer) error {
	return writeVariableLength(w, a.Bytes())
}

func (k *PublicKey) Unmarshal(r Reader) error {
	return readExpectedLength(r, k[:], "PublicKey")
}

func (k *PublicKey) Marshal(w io.Writer) error {
	if k.IsZero() {
		return writeVariableLength(w, []byte(nil))
	}
	return writeVariableLength(w, k.Bytes())
}

func (k *RegularKey) Unmarshal(r Reader) error {
	return readExpectedLength(r, k[:], "RegularKey")
}

func (k *RegularKey) Marshal(w io.Writer) error {
	return writeVariableLength(w, k.Bytes())
}

func (p *PathSet) Unmarshal(r Reader) error {
	for i := 0; ; i++ {
		*p = append(*p, Path{})
		for b, err := r.ReadByte(); ; b, err = r.ReadByte() {
			entry := pathEntry(b)
			if entry == PATH_BOUNDARY {
				break
			}
			if err != nil {
				return err
			}
			if entry == PATH_END {
				return nil
			}
			var pe PathElem
			if entry&PATH_ACCOUNT > 0 {
				pe.Account = new(Account)
				if _, err := r.Read(pe.Account.Bytes()); err != nil {
					return err
				}
			}
			if entry&PATH_CURRENCY > 0 {
				pe.Currency = new(Currency)
				if _, err := r.Read(pe.Currency.Bytes()); err != nil {
					return err
				}
			}
			if entry&PATH_ISSUER > 0 {
				pe.Issuer = new(Account)
				if _, err := r.Read(pe.Issuer.Bytes()); err != nil {
					return err
				}
			}
			(*p)[i] = append((*p)[i], pe)
		}
	}
}

func (p *PathSet) Marshal(w io.Writer) error {
	for i, path := range *p {
		for _, entry := range path {
			if err := write(w, entry.pathEntry()); err != nil {
				return err
			}
			if err := write(w, entry.Account.Bytes()); err != nil {
				return err
			}
			if err := write(w, entry.Currency.Bytes()); err != nil {
				return err
			}
			if err := write(w, entry.Issuer.Bytes()); err != nil {
				return err
			}
		}
		var err error
		if i < len(*p)-1 {
			err = write(w, PATH_BOUNDARY)
		} else {
			err = write(w, PATH_END)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (res *TransactionResult) Marshal(w io.Writer) error {
	if *res > math.MaxUint8 || *res < 0 {
		return fmt.Errorf("Cannot marshal transaction result: %d", *res)
	}
	return write(w, uint8(*res))
}

func (res *TransactionResult) Unmarshal(r Reader) error {
	var result uint8
	if err := binary.Read(r, binary.BigEndian, &result); err != nil {
		return err
	}
	*res = TransactionResult(result)
	return nil
}

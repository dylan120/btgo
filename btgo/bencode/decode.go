package bencode

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
)

type Unmarshaler interface {
	UnmarshalBencode([]byte) error
}

var unmarshaler = reflect.TypeOf(func() *Unmarshaler {
	var i Unmarshaler
	return &i
}()).Elem()

type Decoder struct {
	r   *bufio.Reader
	buf bytes.Buffer
	off int
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

func Unmarshal(data []byte, v interface{}) (err error) {
	buf := bytes.NewBuffer(data)
	d := NewDecoder(buf)
	err = d.decode(reflect.ValueOf(v))
	return
}

func (d *Decoder) readValue() (b []byte, err error) {
	ch, err := d.r.Peek(1)
	if err != nil {
		return nil, err
	}
	if ch[0] == 'e' {
		b, err := d.r.ReadByte()
		if err != nil {
			return nil, errors.New("read e")
		}

		return []byte{b}, nil
	}

	switch ch[0] {
	case 'i':
		i, err := d.r.ReadBytes('e')
		if err != nil {
			return nil, err
		}
		b = append(b, i...)

	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		s, err := d.r.ReadBytes(':')
		if err != nil {
			return nil, err
		}
		length, err := strconv.ParseInt(string(s[0:len(s)-1]), 10, 0)
		b1, err := d.readNBytes(int(length))
		b = append(b, s...)
		b = append(b, b1...)

	case 'l', 'd':
		b1, err := d.r.ReadByte()
		if err != nil {
			return nil, errors.New("end")
		}
		b = append(b, b1)
		for {
			b2, err := d.readValue()
			if err != nil {
				break
			}
			b = append(b, b2...)
			if len(b2) == 1 && b2[0] == 'e' {
				break
			}
		}

	default:
		return nil, errors.New("unsupported type" + string(ch) + "xxxx")
	}
	return b, nil
}

func (d *Decoder) decodeUnmarshaler(v reflect.Value) bool {
	if !v.Type().Implements(unmarshaler) {
		if v.Addr().Type().Implements(unmarshaler) {
			v = v.Addr()
		} else {
			return false
		}
	}
	m := v.Interface().(Unmarshaler)
	data, err := d.readValue()
	err = m.UnmarshalBencode(data)
	if err != nil {
		fmt.Println(err)
	}
	return true
}

func (d *Decoder) readNBytes(n int) (r []byte, err error) {
	var b byte
	for i := 0; i < n; i++ {
		b, err = d.r.ReadByte()
		if err != nil {
			break
		}
		r = append(r, b)
	}
	return
}

func (d *Decoder) decodeInt(v reflect.Value) (err error) {
	data, err := d.r.ReadBytes('e')
	if err != nil {
		return err
	}

	s := string(data[1 : len(data)-1])
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return err
		}
		if v.OverflowInt(n) {
			return errors.New("overflow int")
		}
		v.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		if v.OverflowUint(n) {
			return errors.New("overflow uint")
		}
		v.SetUint(n)
	case reflect.Bool:
		v.SetBool(s != "0")
	}

	return
}

func (d *Decoder) decodeString(v reflect.Value) (err error) {
	data, err := d.r.ReadBytes(':')
	if err != nil {
		return err
	}
	length, err := strconv.ParseInt(string(data[0:len(data)-1]), 10, 0)
	s, err := d.readNBytes(int(length))
	if err != nil {
		return err
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(string(s))
		return nil
	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			v.SetBytes(s)
			return nil
		} else {
			return errors.New("unexpected type")
		}
	case reflect.Array:
		if v.Type().Elem().Kind() != reflect.Uint8 {
			break
		}
		reflect.Copy(v, reflect.ValueOf(s))
		return nil
	}

	return errors.New("decode string failed")
}

func (d *Decoder) decodeDict() (err error) {
	b, err := d.r.ReadByte()
	if err != nil {
		if b == 'e' {
			_, err := d.r.ReadByte()
			return err
		}
	}

	for {
		ch, err := d.r.Peek(1)
		if err != nil {
			return err
		}

		if ch[0] == 'e' {
			d.r.ReadByte()
			break
		}

		data, err := d.r.ReadBytes(':')
		if err != nil {
			return err
		}
		length, err := strconv.ParseInt(string(data[0:len(data)-1]), 10, 0)
		_, err = d.readNBytes(int(length))
		if err != nil {
			fmt.Println(">>>>", err)
			return err
		}

		ch, err = d.r.Peek(1)
		if err != nil {
			return err
		}

		switch ch[0] {
		case 'i':
			var i int
			err = d.decodeInt(reflect.ValueOf(&i).Elem())
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			var s string
			err = d.decodeString(reflect.ValueOf(&s).Elem())
		case 'd':
			err = d.decodeDict()
		}
	}

	return
}

func (d *Decoder) decodeList(v reflect.Value) (err error) {
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
	default:
		return errors.New("invaild list")
	}
	_, err = d.r.ReadByte()
	if err != nil {
		return err
	}

	i := 0
	for ; ; i++ {
		ch, err := d.r.Peek(1)
		if err != nil {
			break
		}
		if ch[0] == 'e' {
			d.r.ReadByte()
			break
		}

		if v.Kind() == reflect.Slice && i >= v.Len() {
			v.Set(reflect.Append(v, reflect.Zero(v.Type().Elem())))
		}
		if i < v.Len() {
			err = d.decode(v.Index(i))
		}
	}

	if i < v.Len() {
		if v.Kind() == reflect.Array {
			z := reflect.Zero(v.Type().Elem())
			for n := v.Len(); i < n; i++ {
				v.Index(i).Set(z)
			}
		} else {
			v.SetLen(i)
		}
	}

	if i == 0 && v.Kind() == reflect.Slice {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}
	return err
}

func (d *Decoder) getStrcuctMap(m map[string]reflect.Value, v reflect.Value) {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		tfield := t.Field(i)
		tag := getTag(tfield.Tag)
		f := v.FieldByIndex(tfield.Index)
		key := tag.Key()
		if key == "" {
			key = tfield.Name
		}
		m[key] = f
	}
}

func (d *Decoder) decodeUnexpetedValue() (err error) {
	ch, err := d.r.Peek(1)
	if err != nil {
		return err
	}
	switch ch[0] {
	case 'i':
		var i int
		err = d.decodeInt(reflect.ValueOf(&i).Elem())
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		var s string
		err = d.decodeString(reflect.ValueOf(&s).Elem())
	case 'l':
		d.r.ReadByte()
		if ch[0] == 'e' {
			d.r.ReadByte()
			return err
		}
		err = d.decodeUnexpetedValue()
	case 'd':
		err = d.decodeDict()

	default:
		return errors.New("unsupported type" + string(ch) + "xxxx")
	}
	ch, err = d.r.Peek(1)
	if err != nil {
		return err
	}

	return err
}

func (d *Decoder) decodeStruct(v reflect.Value) (err error) {

	b, err := d.r.ReadByte()
	if err != nil {
		if b == 'e' {
			_, err := d.r.ReadByte()
			return err
		}
	}
	m := make(map[string]reflect.Value)
	d.getStrcuctMap(m, v)

	for {
		ch, err := d.r.Peek(1)
		if err != nil {
			return err
		}

		if ch[0] == 'e' {
			d.r.ReadByte()
			break
		}

		data, err := d.r.ReadBytes(':')
		if err != nil {
			return err
		}
		length, err := strconv.ParseInt(string(data[0:len(data)-1]), 10, 0)
		s, err := d.readNBytes(int(length))
		if err != nil {
			fmt.Println(">>>>", err)
			return err
		}

		mkey := string(s)

		val, ok := m[mkey] //TODO
		//fmt.Println(mkey,ok)
		if ok {
			err = d.decode(val)
			if err != nil {
				return err
			}
		} else {
			d.decodeUnexpetedValue()
		}
		//dd,_:=d.r.Peek(1)
		//fmt.Println(mkey, ok,val.Kind())
	}
	//dd,_:=d.r.Peek(20)
	//fmt.Println("wewewewewewewewewewwwwwwwwwwwwwwwwww",string(dd))
	return err
}

func (d *Decoder) indirect(v reflect.Value) reflect.Value {
	for {
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}
		if v.Elem().Kind() != reflect.Ptr && v.CanSet() {
			v = v.Elem()
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

func (d *Decoder) decodeInterface(v reflect.Value) (i interface{}, err error) {
	ch, err := d.r.Peek(1)
	if err != nil {
		return nil, err
	}
	//fmt.Println("decodeInterface")
	switch ch[0] {
	case 'i':
		b, err := d.r.ReadBytes('e')
		if err != nil {
			return nil, err
		}
		s := b[1 : len(b)-1]
		integer,err :=strconv.ParseInt(string(s), 10, 0)
		if err != nil {
			return nil, err
		}
		return integer, nil

	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		b, err := d.r.ReadBytes(':')
		if err != nil {
			return nil, err
		}
		length, err := strconv.ParseInt(string(b[0:len(b)-1]), 10, 0)
		s, err := d.readNBytes(int(length))
		if err != nil {
			return nil, err
		}
		//v.SetBytes(s)
		return string(s),nil
	case 'l':
		var ifSlice []interface{}
		b, err := d.r.ReadByte()
		if err != nil {
			return nil,err
		}
		if b != 'e' {
			for {
				ch, err := d.r.Peek(1)
				if err != nil {
					return nil, err
				}
				if ch[0] == 'e'{
					break
				}

				if_, err := d.decodeInterface(v)
				if err != nil {
					return nil, err
				}
				ifSlice = append(ifSlice, if_)
			}
		}
		if ifSlice == nil {
			ifSlice = make([]interface{}, 0, 0)
		}
		v.Set(reflect.ValueOf(ifSlice))
	case 'd':
	}
	return nil, err
}

func (d *Decoder) decode(v reflect.Value) (err error) {
	val := d.indirect(v)
	if d.decodeUnmarshaler(val) {
		return err
	}

	if val.Kind() == reflect.Interface {
		d.decodeInterface(val)
		return
	}

	ch, err := d.r.Peek(1)
	if err != nil {
		return err
	}

	switch ch[0] {
	case 'i':
		err = d.decodeInt(val)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		err = d.decodeString(val)
	case 'l':
		err = d.decodeList(val)
	case 'd':
		err = d.decodeStruct(val)
	default:
		err = errors.New("invalid input")
	}
	return err
}

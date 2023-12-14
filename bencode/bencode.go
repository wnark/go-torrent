package bencode

import (
	"bufio"
	"errors"
	"io"
)

var (
	ErrNum = errors.New("expect num")
	ErrCol = errors.New("expect colon")
	ErrEpI = errors.New("expect char i")
	ErrEpE = errors.New("expect char e")
	ErrTyp = errors.New("wrong type")
	ErrIvd = errors.New("invalid bencode")
)

// 定义8位的BType
type BType uint8

// 定义四种类型就够了
const (
	BSTR  BType = 0x01
	BINT  BType = 0x02
	BLIST BType = 0x03
	BDICT BType = 0x04
)
// 没有泛型支持，在这里定义了空接口
type BValue interface{}
// 完整的Object是包含了type和value,是有对应关系的.
type BObject struct {
	type_ BType
	val_  BValue
}
// BObject 在 go语言怎么在代码中表示
func (o *BObject) Str() (string, error) {
	// 如果type是string，那value就应该是string
	if o.type_ != BSTR {
		return "", ErrTyp
	}
	return o.val_.(string), nil
}

func (o *BObject) Int() (int, error) {
	// 如果type是int，那value就应该是int
	if o.type_ != BINT {
		return 0, ErrTyp
	}
	return o.val_.(int), nil
}

func (o *BObject) List() ([]*BObject, error) {
	// 如果type是list,那value就应该是slice指针
	if o.type_ != BLIST {
		return nil, ErrTyp
	}
	return o.val_.([]*BObject), nil
}

func (o *BObject) Dict() (map[string]*BObject, error) {
	// 如果type是dict,那key为string,value为BObject指针的map,Go语言中的map是引用类型，必须初始化才能使用.
	if o.type_ != BDICT {
		return nil, ErrTyp
	}
	return o.val_.(map[string]*BObject), nil
}

// 进行 BObject 到 Bencode 的序列化,变成字符写到iowriter中,返回写了多少长度int
func (o *BObject) Bencode(w io.Writer) int {
	bw, ok := w.(*bufio.Writer)
	if !ok {
		bw = bufio.NewWriter(w)
	}
	wLen := 0
	// 根据type进行switch
	switch o.type_ {
	case BSTR:
		str, _ := o.Str()
		wLen += EncodeString(bw, str)
	case BINT:
		val, _ := o.Int()
		wLen += EncodeInt(bw, val)
	case BLIST:
		bw.WriteByte('l')
		list, _ := o.List()
		// 遍历
		for _, elem := range list {
			// 递归调用
			wLen += elem.Bencode(bw)
		}
		bw.WriteByte('e')
		wLen += 2
	case BDICT:
		bw.WriteByte('d')
		dict, _ := o.Dict()
		for k, v := range dict {
			// 递归调用进行序列化
			wLen += EncodeString(bw, k)
			wLen += v.Bencode(bw)
		}
		// 序列化完成后，可以补上e
		bw.WriteByte('e')
		wLen += 2
	}
	// 序列化完成后，flush一下
	bw.Flush()
	// 累加出来的wlen就是往write内写的总长度
	return wLen
}

func checkNum(data byte) bool {
	return data >= '0' && data <= '9'
}

func readDecimal(r *bufio.Reader) (val int, len int) {
	sign := 1
	b, _ := r.ReadByte()
	len++
	if b == '-' {
		sign = -1
		b, _ = r.ReadByte()
		len++
	}
	for {
		if !checkNum(b) {
			r.UnreadByte()
			len--
			return sign * val, len
		}
		val = val*10 + int(b-'0')
		b, _ = r.ReadByte()
		len++
	}
}

// 处理int的函数
func writeDecimal(w *bufio.Writer, val int) (len int) {
	// 如果输入为0的话，输出为0就结束了
	if val == 0 {
		w.WriteByte('0')
		len++
		return
	}
	// 如果不是0是复数，则输出负号，并将其转为正数
	if val < 0 {
		w.WriteByte('-')
		len++
		val *= -1
	}
	// 不断取商和余，将十进制的数字转化为一个个的数，然后转为ascii码
	dividend := 1
	for {
		if dividend > val {
			dividend /= 10
			break
		}
		dividend *= 10
	}
	for {
		num := byte(val / dividend)
		w.WriteByte('0' + num)
		len++
		if dividend == 1 {
			return
		}
		val %= dividend
		dividend /= 10
	}
}

// 编码string: 3:abc
func EncodeString(w io.Writer, val string) int {
	// 求出字符串的长度
	strLen := len(val)
	// 以十进制对应的ascii码的形式写进io.writer
	bw := bufio.NewWriter(w)
	wLen := writeDecimal(bw, strLen)
	bw.WriteByte(':')
	// 长度增加
	wLen++
	// 将长度写入？
	bw.WriteString(val)
	wLen += strLen

	// io请求写完了，刷新下
	err := bw.Flush()
	if err != nil {
		return 0
	}
	return wLen
}

// 解码string 3:abc
func DecodeString(r io.Reader) (val string, err error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	// 解码出前面的数字num,比如3.len为长度
	num, len := readDecimal(br)
	if len == 0 {
		return val, ErrNum
	}
	b, err := br.ReadByte()
	if b != ':' {
		return val, ErrCol
	}
	// 获取长度为num的buffer
	buf := make([]byte, num)
	// 把剩下的读出来
	_, err = io.ReadAtLeast(br, buf, num)
	val = string(buf)
	return
}

// 编码int
func EncodeInt(w io.Writer, val int) int {
	bw := bufio.NewWriter(w)
	wLen := 0
	// 写开头i
	bw.WriteByte('i')
	wLen++
	// 转换int数字
	nLen := writeDecimal(bw, val)
	wLen += nLen
	// 写结尾e
	bw.WriteByte('e')
	wLen++

	err := bw.Flush()
	if err != nil {
		return 0
	}
	return wLen
}
// 解码int
func DecodeInt(r io.Reader) (val int, err error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	
	b, err := br.ReadByte()
	// 读取开头i
	if b != 'i' {
		return val, ErrEpI
	}
	// 将byte组装成int
	val, _ = readDecimal(br)
	b, err = br.ReadByte()
	// 读取结尾e
	if b != 'e' {
		return val, ErrEpE
	}
	return
}

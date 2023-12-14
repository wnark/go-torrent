package bencode

import (
	"bufio"
	"io"
)
// 反序列化 Bencode 文本 转成 BObject，实现解析
func Parse(r io.Reader) (*BObject, error) {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	//recursive descent parsing
	// 拿到第一位，但不读，只看一眼是什么东西
	b, err := br.Peek(1)
	if err != nil {
		return nil, err
	}
	var ret BObject
	switch {
	// 如果第一位是数字的话，就是string的处理方式 
	case b[0] >= '0' && b[0] <= '9':
		// parse string
		// 解码后
		val, err := DecodeString(br)
		if err != nil {
			return nil, err
		}
		// 将类型和值设置进去
		ret.type_ = BSTR
		ret.val_ = val
	// 如果是i，则是int的逻辑
	case b[0] == 'i':
		// parse int
		// 解码后
		val, err := DecodeInt(br)
		if err != nil {
			return nil, err
		}
		// 将类型和值设置进去
		ret.type_ = BINT
		ret.val_ = val
	// 如果是l，则是list的逻辑
	case b[0] == 'l':
		// parse list
		// 把第一位l读出来处理掉
		br.ReadByte()
		var list []*BObject
		for {
			// 循环处理,如果是e的话读到最后一位,这里的e不可能是子元素的e，因为他会通过Parse函数把e pass掉?
			if p, _ := br.Peek(1); p[0] == 'e' {
				br.ReadByte()
				break
			}
			// 通过递归调用Parse,获取elem则为BObject的指针
			elem, err := Parse(br)
			if err != nil {
				return nil, err
			}
			// 添加到结果的list中去
			list = append(list, elem)
		}
		// 将类型和值设置进去
		ret.type_ = BLIST
		ret.val_ = list
	// 如果是d,则是dict的逻辑
	case b[0] == 'd':
		// parse map
		br.ReadByte()
		dict := make(map[string]*BObject)
		for {
			// 和list一样判断解析是否结束了
			if p, _ := br.Peek(1); p[0] == 'e' {
				br.ReadByte()
				break
			}
			// 先把key解析出来
			key, err := DecodeString(br)
			if err != nil {
				return nil, err
			}
			// 获取BObject的指针
			val, err := Parse(br)
			if err != nil {
				return nil, err
			}
			// 设置key和value
			dict[key] = val
		}
		// 将类型和值设置进去
		ret.type_ = BDICT
		ret.val_ = dict
	default:
		return nil, ErrIvd
	}
	return &ret, nil
}

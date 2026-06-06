---
weight: 534
title: "如何实现字符串和byte切片的零拷贝转换"
slug: /zero-conv
---

这是一个非常精典的例子。实现字符串和 bytes 切片之间的转换，要求是 `zero-copy`。想一下，一般的做法，都需要遍历字符串或 bytes 切片，再挨个赋值。

完成这个任务，我们需要了解 slice 和 string 的底层数据结构：

```golang
type StringHeader struct {
	Data uintptr
	Len  int
}

type SliceHeader struct {
	Data uintptr
	Len  int
	Cap  int
}
```

上面是反射包下的结构体，路径：src/reflect/value.go。只需要共享底层 Data 和 Len 就可以实现 `zero-copy`。

```golang
func string2bytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}
func bytes2string(b []byte) string{
	return *(*string)(unsafe.Pointer(&b))
}
```

原理上是利用指针的强转，代码比较简单，不作详细解释。

## Go 1.20 之后的写法

需要注意的是，上面这种依赖 `reflect.StringHeader` / `reflect.SliceHeader` 内存布局的写法在新版本里已经不推荐使用了：自 Go 1.20 起，`reflect.StringHeader` 和 `reflect.SliceHeader` 都被标记为 **Deprecated**。同时，标准库 `unsafe` 包从 Go 1.17 起提供了 `unsafe.Slice`，从 Go 1.20 起又新增了 `unsafe.String` 和 `unsafe.StringData`，可以更安全、更清晰地完成零拷贝转换：

```golang
import "unsafe"

// string -> []byte
func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// []byte -> string
func BytesToString(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}
```

这种写法的好处是：不再依赖 `Header` 结构体的字段布局，由编译器/运行时保证指针与长度的对应关系，可读性也更好。

几点提醒：

- `unsafe.StringData` 作用在空字符串 `""` 上时，返回的指针是不确定的（unspecified），因此 `StringToBytes` 得到的切片只应在 `len(s) > 0` 时使用。
- 如果用 `unsafe.String(&b[0], len(b))` 这种取首元素地址的写法（源自 `strings.Clone`），当 `b` 为空切片时 `&b[0]` 会越界 panic，需要先判断长度：

```golang
func BytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(&b[0], len(b))
}
```

  相比之下，使用 `unsafe.SliceData(b)`（Go 1.20+）则能正确处理空切片，无需额外判断。

- 无论哪种写法，本质都是“零拷贝”地共享底层内存，因此 **必须保证转换得到的结果不被写入**（尤其是 `string -> []byte` 之后再写这块内存，会破坏字符串不可变的假设，属于未定义行为），并且原始数据的生命周期要覆盖结果的使用周期。
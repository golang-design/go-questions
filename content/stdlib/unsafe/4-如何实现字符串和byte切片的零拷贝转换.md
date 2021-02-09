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
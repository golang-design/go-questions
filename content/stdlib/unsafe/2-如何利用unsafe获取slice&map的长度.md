---
weight: 532
title: "如何利用unsafe获取slice&map的长度"
slug: /len
---

# 获取 slice 长度
通过前面关于 slice 的[文章](https://mp.weixin.qq.com/s/MTZ0C9zYsNrb8wyIm2D8BA)，我们知道了 slice header 的结构体定义：

```golang
// runtime/slice.go
type slice struct {
    array unsafe.Pointer // 元素指针
    len   int // 长度 
    cap   int // 容量
}
```

调用 make 函数新建一个 slice，底层调用的是 makeslice 函数，返回的是 slice 结构体：

```golang
func makeslice(et *_type, len, cap int) slice
```

因此我们可以通过 unsafe.Pointer 和 uintptr 进行转换，得到 slice 的字段值。

```golang
func main() {
	s := make([]int, 9, 20)
	var Len = *(*int)(unsafe.Pointer(uintptr(unsafe.Pointer(&s)) + uintptr(8)))
	fmt.Println(Len, len(s)) // 9 9

	var Cap = *(*int)(unsafe.Pointer(uintptr(unsafe.Pointer(&s)) + uintptr(16)))
	fmt.Println(Cap, cap(s)) // 20 20
}
```

Len，cap 的转换流程如下：

```golang
Len: &s => pointer => uintptr => pointer => *int => int
Cap: &s => pointer => uintptr => pointer => *int => int
```

# 获取 map 长度
再来看一下上篇文章我们讲到的 map：

```golang
type hmap struct {
	count     int
	flags     uint8
	B         uint8
	noverflow uint16
	hash0     uint32

	buckets    unsafe.Pointer
	oldbuckets unsafe.Pointer
	nevacuate  uintptr

	extra *mapextra
}
```

和 slice 不同的是，makemap 函数返回的是 hmap 的指针，注意是指针：

```golang
func makemap(t *maptype, hint int64, h *hmap, bucket unsafe.Pointer) *hmap
```

我们依然能通过 unsafe.Pointer 和 uintptr 进行转换，得到 hamp 字段的值，只不过，现在 count 变成二级指针了：

```golang
func main() {
	mp := make(map[string]int)
	mp["qcrao"] = 100
	mp["stefno"] = 18

	count := **(**int)(unsafe.Pointer(&mp))
	fmt.Println(count, len(mp)) // 2 2
}
```

count 的转换过程：

```golang
&mp => pointer => **int => int
```

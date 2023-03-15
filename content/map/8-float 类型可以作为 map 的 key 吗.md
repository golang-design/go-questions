---
weight: 208
title: "float 类型可以作为 map 的 key 吗"
slug: /float-as-key
---

从语法上看，是可以的。Go 语言中只要是可比较的类型都可以作为 key。除开 slice，map，functions 这几种类型，其他类型都是 OK 的。具体包括：布尔值、数字、字符串、指针、通道、接口类型、结构体、只包含上述类型的数组。这些类型的共同特征是支持 `==` 和 `!=` 操作符，`k1 == k2` 时，可认为 k1 和 k2 是同一个 key。如果是结构体，只有 hash 后的值相等以及字面值相等，才被认为是相同的 key。很多字面值相等的，hash出来的值不一定相等，比如引用。

顺便说一句，任何类型都可以作为 value，包括 map 类型。

来看个例子：

```golang
func main() {
	m := make(map[float64]int)
	m[1.4] = 1
	m[2.4] = 2
	m[math.NaN()] = 3
	m[math.NaN()] = 3

	for k, v := range m {
		fmt.Printf("[%v, %d] ", k, v)
	}

	fmt.Printf("\nk: %v, v: %d\n", math.NaN(), m[math.NaN()])
	fmt.Printf("k: %v, v: %d\n", 2.400000000001, m[2.400000000001])
	fmt.Printf("k: %v, v: %d\n", 2.4000000000000000000000001, m[2.4000000000000000000000001])

	fmt.Println(math.NaN() == math.NaN())
}
```

程序的输出：

```shell
[2.4, 2] [NaN, 3] [NaN, 3] [1.4, 1] 
k: NaN, v: 0
k: 2.400000000001, v: 0
k: 2.4, v: 2
false
```

例子中定义了一个 key 类型是 float 型的 map，并向其中插入了 4 个 key：1.4， 2.4， NAN，NAN。

打印的时候也打印出了 4 个 key，如果你知道 NAN != NAN，也就不奇怪了。因为他们比较的结果不相等，自然，在 map 看来就是两个不同的 key 了。

接着，我们查询了几个 key，发现 NAN 不存在，2.400000000001 也不存在，而 2.4000000000000000000000001 却存在。

有点诡异，不是吗？

接着，我通过汇编发现了如下的事实：

当用 float64 作为 key 的时候，先要将其转成 uint64 类型，再插入 key 中。

具体是通过 `Float64frombits` 函数完成：

```golang
// Float64frombits returns the floating point number corresponding
// the IEEE 754 binary representation b.
func Float64frombits(b uint64) float64 { return *(*float64)(unsafe.Pointer(&b)) }
```

也就是将浮点数表示成 IEEE 754 规定的格式。如赋值语句：

```asm
0x00bd 00189 (test18.go:9)      LEAQ    "".statictmp_0(SB), DX
0x00c4 00196 (test18.go:9)      MOVQ    DX, 16(SP)
0x00c9 00201 (test18.go:9)      PCDATA  $0, $2
0x00c9 00201 (test18.go:9)      CALL    runtime.mapassign(SB)
```

`"".statictmp_0(SB)` 变量是这样的：

```asm
"".statictmp_0 SRODATA size=8
        0x0000 33 33 33 33 33 33 03 40
"".statictmp_1 SRODATA size=8
        0x0000 ff 3b 33 33 33 33 03 40
"".statictmp_2 SRODATA size=8
        0x0000 33 33 33 33 33 33 03 40
```

我们再来输出点东西：

```golang
package main

import (
	"fmt"
	"math"
)

func main() {
	m := make(map[float64]int)
	m[2.4] = 2

    fmt.Println(math.Float64bits(2.4))
	fmt.Println(math.Float64bits(2.400000000001))
	fmt.Println(math.Float64bits(2.4000000000000000000000001))
}
```

```shell
4612586738352862003
4612586738352864255
4612586738352862003
```

转成十六进制为：

```shell
0x4003333333333333
0x4003333333333BFF
0x4003333333333333
```

和前面的 `"".statictmp_0` 比较一下，很清晰了吧。`2.4` 和 `2.4000000000000000000000001` 经过 `math.Float64bits()` 函数转换后的结果是一样的。自然，二者在 map 看来，就是同一个 key 了。

再来看一下 NAN（not a number）：

```golang
// NaN returns an IEEE 754 ``not-a-number'' value.
func NaN() float64 { return Float64frombits(uvnan) }
```

uvnan 的定义为：

```golang
uvnan    = 0x7FF8000000000001
```

NAN() 直接调用 `Float64frombits`，传入写死的 const 型变量 `0x7FF8000000000001`，得到 NAN 型值。既然，NAN 是从一个常量解析得来的，为什么插入 map 时，会被认为是不同的 key？

这是由类型的哈希函数决定的，例如，对于 64 位的浮点数，它的哈希函数如下：

```golang
func f64hash(p unsafe.Pointer, h uintptr) uintptr {
	f := *(*float64)(p)
	switch {
	case f == 0:
		return c1 * (c0 ^ h) // +0, -0
	case f != f:
		return c1 * (c0 ^ h ^ uintptr(fastrand())) // any kind of NaN
	default:
		return memhash(p, h, 8)
	}
}
```

第二个 case，`f != f` 就是针对 `NAN`，这里会再加一个随机数。

这样，所有的谜题都解开了。

由于 NAN 的特性：

```shell
NAN != NAN
hash(NAN) != hash(NAN)
```

因此向 map 中查找的 key 为 NAN 时，什么也查不到；如果向其中增加了 4 次 NAN，遍历会得到 4 个 NAN。

最后说结论：float 型可以作为 key，但是由于精度的问题，会导致一些诡异的问题，慎用之。

---

关于当 key 是引用类型时，判断两个 key 是否相等，需要 hash 后的值相等并且 key 的字面量相等。由 @WuMingyu 补充的例子：

```go
func TestT(t *testing.T) {
	type S struct {
		ID	int
	}
	s1 := S{ID: 1}
	s2 := S{ID: 1}

	var h = map[*S]int {}
	h[&s1] = 1
	t.Log(h[&s1])
	t.Log(h[&s2])
	t.Log(s1 == s2)
}
```
test output:
```shell
=== RUN   TestT
--- PASS: TestT (0.00s)
    endpoint_test.go:74: 1
    endpoint_test.go:75: 0
    endpoint_test.go:76: true
PASS

Process finished with exit code 0
```

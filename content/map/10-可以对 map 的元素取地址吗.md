---
weight: 210
title: "可以对 map 的元素取地址吗"
slug: /element-address
---

无法对 map 的 key 或 value 进行取址。以下代码不能通过编译：

```golang
package main

import "fmt"

func main() {
	m := make(map[string]int)

	fmt.Println(&m["qcrao"])
}
```

编译报错：

```shell
./main.go:8:14: cannot take the address of m["qcrao"]
```

如果通过其他 hack 的方式，例如 unsafe.Pointer 等获取到了 key 或 value 的地址，也不能长期持有，因为一旦发生扩容，key 和 value 的位置就会改变，之前保存的地址也就失效了。

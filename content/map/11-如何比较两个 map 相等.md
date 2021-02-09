---
weight: 211
title: "如何比较两个 map 相等"
slug: /compare
---


map 深度相等的条件：

```shell
1、都为 nil
2、非空、长度相等，指向同一个 map 实体对象
3、相应的 key 指向的 value “深度”相等
```

直接将使用 map1 == map2 是错误的。这种写法只能比较 map 是否为 nil。

```golang
package main

import "fmt"

func main() {
	var m map[string]int
	var n map[string]int

	fmt.Println(m == nil)
	fmt.Println(n == nil)

	// 不能通过编译
	//fmt.Println(m == n)
}
```

输出结果：

```golang
true
true
```

因此只能是遍历map 的每个元素，比较元素是否都是深度相等。


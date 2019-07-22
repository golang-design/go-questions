GoRoot 是  Go 的安装路径。mac 或 unix 是在 `/usr/local/go` 路径上，来看下这里都装了些什么：

![/usr/local/go](https://user-images.githubusercontent.com/7698088/60344492-41178180-99e9-11e9-98b0-b1f8d64ce97d.png)

bin 目录下面：

![bin](https://user-images.githubusercontent.com/7698088/60344698-b5522500-99e9-11e9-8883-a5bf2460fba0.png)

pkg 目录下面：

![pkg](https://user-images.githubusercontent.com/7698088/60344731-c7cc5e80-99e9-11e9-8002-83f3debc09a6.png)

Go 工具目录如下，其中比较重要的有编译器 `compile`，链接器 `link`：

![pkg/tool](https://user-images.githubusercontent.com/7698088/60379164-888d2480-9a60-11e9-9322-920c0e1b2b3d.png)

GoPath 的作用在于提供一个可以寻找 `.go` 源码的路径，它是一个工作空间的概念，可以设置多个目录。Go 官方要求，GoPath 下面需要包含三个文件夹：

```shell
src
pkg
bin
```

src 存放源文件，pkg 存放源文件编译后的库文件，后缀为 `.a`；bin 则存放可执行文件。
# gossh，实现简单的服务器ssh登录管理以及sftp的文件上传和下载


gossh 支持mac、linux、windows(todo);

## 功能说明


*   管理服务器
*   免输入密码登录服务器
*   sftp文件拷贝，支持本地文件或文件夹拷贝到服务器、服务器文件或文件夹拷贝到本地、支持文件或文件夹多服务器拷贝;支持多线程文件下载，支持文件断点续传。
*   忽略非必要下载文件（todo）

#### 管理服务器

***

*   在安装目录创建配置文件 `config.yml` ，全局的配置文件。
*   在相对目录创建配置文件 `config.yml`，如果在相对目录执行gossh的时候，会优先找当前目录下的配置文件。
*   参数带配置文件 `gossh -c 配置文件.yml`&#x20;
*   字段说明

        addresss:
          - name: '服务器登录名'
            ip: '服务器ip'
            password: '登录服务器密码'
            port: 22 //服务器端口
            key: '登录服务器的缩写'
            pem: '本地密钥 绝对路径'
            servername: '服务器名'
          - name: '服务器登录名'
            ip: '服务器ip'
            password: '登录服务器密码'
            port: 22 //服务器端口
            key: '登录服务器的缩写'
            pem: '本地密钥'
            servername: '服务器名'

#### 免密码登录服务器

***

```bash
gossh link [key|索引]
例：
gossh link 缩写
gossh link 索引
```

#### 文件拷贝

***

```bash
gossh cp 本地文件 key1:远程目录 key2:远程目录....
gossh cp -r 本地文件 key1:远程目录 key2:远程目录... // -r 强制覆盖
gossh cp -t 10 本地文件 key1:远程目录 key2:远程目录.... // -t 携程数量，默认10个携程
```

#### 目录拷贝

***

```bash
gossh cp -d 本地文件夹 key1:远程目录 key2:远程目录....
gossh cp -r -d 本地文件夹 key1:远程目录 key2:远程目录... // -r 强制覆盖
gossh cp -t 10 本地文件夹 key1:远程目录 key2:远程目录.... // -t 携程数量，默认10个携程
```


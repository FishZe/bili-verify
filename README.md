# Bili Verify B站用户认证工具

#### 通过直播间弹幕的形式, 用户发送验证码让, 通过认证。

本项目可以在不经过官方授权的情况下, 实现B站用户认证, 实现账号绑定等功能, 可用于B站用户的身份认证。

## 如何使用

### 1. 准备工作: 前往[`https://verify.fishze.top/login`](https://verify.fishze.top/login), 通过`Github`授权认证

前往上述链接, 授权登录`bilibili verify`的`Github OAuth App`后, 您将会被重定向到`https://verify.fishze.top/login/redirect`

此时, 会向您返回如下内容:
```json
{
	"code": 0,
	"data": {
		"email": "contact.github@fishze.top",
		"name": "FishZe",
		"uuid": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	}
}
```
请记录下`data`中`uuid`的值, 该值将会在后续的认证过程中使用到.

当然了, 如果忘记也没关系, 重新访问链接, 会找回之前的`uuid`.

### 2. 发送申请验证请求

> `https://verify.fishze.top/verify/new_verify`

方式: POST

鉴权方式: 在请求头中添加`Authorization`字段, 内容为上文的`uuid`

返回值:
```json
{
  "code": 0,
  "data": {
    "queryId": "f3a4434c-e419-4e7a-a5de-664a3d458a86",
    "roomId": 26703269,
    "roomUrl": "https://live.bilibili.com/26703269",
    "text": "请打开链接: https://live.bilibili.com/26703269 , 在直播间内发送弹幕：R4TEXD1C",
    "verifyMsg": "R4TEXD1C"
  }
}
```
| 字段         | 说明          |
|------------|-------------|
| queryId    | 用于查询验证结果的ID |
| roomId     | 直播间ID       |
| roomUrl    | 直播间链接       |
| text       | 验证提示文本      |
| verifyMsg  | 验证码         |

请让用户打开`roomUrl`中的链接, 并在直播间内发送弹幕`verifyMsg`中的内容.

### 3. 轮询验证结果

> `https://verify.fishze.top/verify/query_verify`
>
方式: POST

鉴权方式: 在请求头中添加`Authorization`字段, 内容为上文的`uuid`

数据内容: obj: `{"queryId": "f3a4434c-e419-4e7a-a5de-664a3d458a86"}`, 其中`queryId`为上文的`queryId`

返回值:

#### 用户已发送时:
```json
{
  "code": 0,
  "data": {
    "error": "",
    "queryId": "f3a4434c-e419-4e7a-a5de-664a3d458a86",
    "userInfo": {
      "uid": 208259,
      "name": "陈睿",
      "medal": ""
    }
  }
}

```
| 字段         | 说明          |
|------------|-------------|
| error      | 错误信息        |
| queryId    | 用于查询验证结果的ID |
| userInfo   | 用户信息        |

其中, 用户信息字段含义如下:

| 字段           | 说明           |
|--------------|--------------|
| uid          | 用户UID        |
| name         | 用户名          |
| medal        | 佩戴勋章         |

#### 用户未发送时:
```json
{
  "code": 1,
  "data": {
    "error": "verify code not used",
    "queryId": "f3a4434c-e419-4e7a-a5de-664a3d458a86",
    "userInfo": {
      "uid": 0,
      "name": "",
      "medal": ""
    }
  }
}
```
### 过期时间

1. 用户弹幕过期时间为`5分钟`
2. 用户发送验证码后, 验证结果的过期时间为`5分钟`
3. 获取验证结果后, 用户信息的过期时间为`5分钟`, 即`5分钟`内再次获取验证结果, 将会返回相同的用户信息

### 频率限制

1. 单个用户每秒请求限制为`50次`
2. 总请求限制在`1000次/秒`

### 错误码

| 错误码  | 说明                   | 备注               |
|------|----------------------|------------------|
| 0    | SuccessCode          | 请求成功             |
| 1    | VerifyCodeNotUsed    | 用户未发送验证码         | 
| 2    | VerifyCodeNotFound   | 未找到此查询ID, 可能是已过期 |
| 3    | VerifyCodeEmpty      | 查询ID为空           |
| 4    | ServerErrorCode      | 服务器错误            |
| 5    | AuthorizationError   | 鉴权错误             |

## 如何部署

### 1. 前往`release`页面下载最新版本

如果没有你的平台, 请提一个issue

### 2. 运行程序, 生成`config.json`

```yaml
port: 8080
room_id: 000000
base_url: https://verify.example.com
need_auth: true
client_id: xxxxxxxxxx
client_secret: xxxxxxxxxxxxxxxxxxx
```
| 字段             | 说明                      |
|----------------|-------------------------|
| port           | 服务端口                    |
| room_id        | 直播间ID                   |
| base_url       | 服务地址                    |
| need_auth      | 是否需要鉴权                  |
| client_id      | Github OAuth App ID     |
| client_secret  | Github OAuth App Secret |

### 3. 重新运行程序

### 4. 有需要可自行配置反向代理

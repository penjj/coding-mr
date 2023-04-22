# cmr
Coding Merge Request
这是一个用于在 Coding 上创建代码合并请求的命令行工具，减少发起多个分支合并请求操作网页的时间。

Install
```bash
# for frontend developer
npm install coding-mr -g

# for unix/linux
curl -sSL https://raw.githubusercontent.com/penjj/coding-mr/master/install.sh | bash

# for windows powershell (untested)
Enable-PSRemoting -Force
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/penjj/coding-mr/master/install.ps1" -OutFile "%UserProfile%\install-cmr.ps1"
powershell.exe -ExecutionPolicy Bypass -File "%UserProfile%\install-cmr.ps1"
```

Usage
```bash
# 设置个人coding openapi token
# 从这里获取你的个人token
# https://coding.net/help/docs/member/tokens.html
git config --global user.codingToken "xxx"

# 设置发起合并的企微机器人回调地址（如果需要），请妥善保管好您的回调地址。
git config --global user.weRobot "xxx"

# 默认使用当前分支作为合并分支
# 参数:
#  -t 合并内容标题
#  -c 合并内容详情
#  -s 需要合并的当前分支，为空则为当前分支
#  -d 需要合并的目标分支，多个用逗号分隔
cmr -t "Merge feature/xxx to develop and release" -d develop,release -c ""
```


#!/bin/bash

executable_name="cmr"
github_url="https://raw.githubusercontent.com/penjj/coding-mr/master/cmr"
target_directory="/usr/local/bin"


# 下载可执行文件并复制到目标目录
echo "Downloading $executable_name from GitHub..."
curl -sSL $github_url -o $executable_name
if [ -f $executable_name ]; then
    chmod +x $executable_name
    echo "Copying $executable_name to $target_directory"
    cp $executable_name $target_directory
    rm $executable_name
    # 添加到 PATH 环境变量中
    if [[ ":$PATH:" != *":$target_directory:"* ]]; then
        echo "Adding $target_directory to PATH"
        export PATH=$PATH:$target_directory
    fi
    echo "Done!"
else
    echo "Error: Failed to download executable from GitHub."
fi
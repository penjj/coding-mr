$executableName = "cmr"
$githubUrl = "https://raw.githubusercontent.com/penjj/coding-mr/master/cmr"
$targetDirectory = "%UserProfile%\cmr"

# 添加到 PATH 环境变量中
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($currentPath -notlike "*$targetDirectory*")
{
    echo "Adding $targetDirectory to PATH"
    $newPath = $currentPath + ";$targetDirectory"
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
}

# 下载可执行文件并复制到目标目录
echo "Downloading $executableName from GitHub..."
$wc = New-Object System.Net.WebClient
$wc.DownloadFile($githubUrl, $executableName)
if (Test-Path $executableName)
{
    echo "Copying $executableName to $targetDirectory"
    Copy-Item $executableName "$targetDirectory\$executableName"
    Remove-Item $executableName
    echo "Done!"
}
else
{
    echo "Error: Failed to download executable from GitHub."
}
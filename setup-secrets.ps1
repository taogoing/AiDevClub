# GitHub Secrets 配置脚本
# 在项目根目录运行: .\setup-secrets.ps1

$REPO = "taogoing/AiDevClub"

Write-Host "=== 配置 GitHub Secrets ===" -ForegroundColor Cyan
Write-Host ""

# 检查 gh 登录状态
$authStatus = gh auth status 2>&1
if ($authStatus -notmatch "Logged in") {
    Write-Host "请先登录 GitHub CLI: gh auth login" -ForegroundColor Red
    exit 1
}

Write-Host "正在配置非敏感 Secrets..." -ForegroundColor Green

# 配置固定值的 secrets
gh secret set SERVER_HOST --body "47.76.151.183" -R $REPO
gh secret set SERVER_USERNAME --body "root" -RFC $REPO
gh secret set PUBLIC_BASE_URL --body "https://aidevclub.xyz" -R $REPO

# 读取 SSH 私钥
$sshKey = Get-Content -Path "deploy_key" -Raw
gh secret set SERVER_SSH_KEY --body $sshKey -R $REPO

Write-Host ""
Write-Host "正在配置敏感 Secrets（请输入密码等信息）..." -ForegroundColor Yellow
Write-Host ""

# 交互式输入敏感信息
$mysqlPassword = Read-Host -Prompt "请输入 MySQL root 密码" -AsSecureString
$mysqlPasswordPlain = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($mysqlPassword)
)

$jwtSecret = Read-Host -Prompt "请输入 JWT 密钥（至少32位）" -AsSecureString
$jwtSecretPlain = [Runtime.InteropServices.Marshal]::PtrToStringAuto(
    [Runtime.InteropServices.Marshal]::SecureStringToBSTR($jwtSecret)
)

$adminEmails = Read-Host -Prompt "请输入管理员邮箱"
$certbotEmail = Read-Host -Prompt "请输入 Certbot 证书邮箱"

gh secret set MYSQL_ROOT_PASSWORD --body $mysqlPasswordPlain -R $REPO
gh secret set JWT_SECRET --body $jwtSecretPlain -R $REPO
gh secret set ADMIN_EMAILS --body $adminEmails -R $REPO
gh secret set CERTBOT_EMAIL --body $certbotEmail -R $REPO

Write-Host ""
Write-Host "=== Secrets 配置完成 ===" -ForegroundColor Green
Write-Host ""
Write-Host "已配置的 Secrets:" -ForegroundColor Cyan
Write-Host "  - SERVER_HOST: 47.76.151.183"
Write-Host "  - SERVER_USERNAME: root"
Write-Host "  - SERVER_SSH_KEY: [已配置]"
Write-Host "  - PUBLIC_BASE_URL: https://aidevclub.xyz"
Write-Host "  - MYSQL_ROOT_PASSWORD: [已配置]"
Write-Host "  - JWT_SECRET: [已配置]"
Write-Host "  - ADMIN_EMAILS: $adminEmails"
Write-Host "  - CERTBOT_EMAIL: $certbotEmail"
Write-Host ""
Write-Host "下一步: 推送代码触发部署" -ForegroundColor Cyan
Write-Host "  git add .github/workflows/deploy.yml"
Write-Host "  git commit -m 'ci: add deployment workflow'"
Write-Host "  git push origin master"

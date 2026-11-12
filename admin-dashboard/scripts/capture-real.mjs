import { chromium } from '@playwright/test'
import { mkdir } from 'node:fs/promises'

const browser = await chromium.launch({ headless: true, executablePath: 'C:/Program Files/Google/Chrome/Application/chrome.exe' })
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 } })
const errors = []
page.on('pageerror', error => errors.push(error.message))
await mkdir('artifacts', { recursive: true })
await page.goto('http://127.0.0.1:5174/login', { waitUntil: 'networkidle' })
await page.locator('#username').fill('admin')
await page.locator('#password').fill('Admin123!')
await page.getByRole('button', { name: /安全登录/ }).click()
await page.waitForURL('**/overview')
await page.waitForTimeout(1500)
await page.goto('http://127.0.0.1:5174/vehicles/LTEST000000000001', { waitUntil: 'networkidle' })
await page.waitForTimeout(2500)
await page.screenshot({ path: 'artifacts/real-vehicle-detail.png', fullPage: true })
await page.getByRole('button', { name: /远程解锁/ }).click()
await page.locator('.confirm-check input').check()
await page.getByRole('button', { name: /确认并发送/ }).click()
await page.getByText('车辆已确认执行成功', { exact: false }).waitFor({ timeout: 15000 })
await page.screenshot({ path: 'artifacts/real-command-succeeded.png', fullPage: true })
console.log(JSON.stringify({ realBackend: true, command: 'succeeded', pageErrors: errors }))
await browser.close()
if (errors.length) process.exitCode = 1

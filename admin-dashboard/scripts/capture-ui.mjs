import { chromium } from '@playwright/test'
import { mkdir } from 'node:fs/promises'

const browser = await chromium.launch({
  headless: true,
  executablePath: 'C:/Program Files/Google/Chrome/Application/chrome.exe',
})
const page = await browser.newPage({ viewport: { width: 1600, height: 1000 }, deviceScaleFactor: 1 })
const errors = []
page.on('pageerror', error => errors.push(error.message))
await mkdir('artifacts', { recursive: true })
await page.goto('http://127.0.0.1:5173/login', { waitUntil: 'networkidle' })
await page.screenshot({ path: 'artifacts/login.png', fullPage: true })
await page.getByRole('button', { name: /登录|进入/ }).click()
await page.waitForURL('**/overview')
await page.waitForTimeout(2500)
await page.screenshot({ path: 'artifacts/overview.png', fullPage: true })
await page.goto('http://127.0.0.1:5173/fleet', { waitUntil: 'networkidle' })
await page.waitForTimeout(1500)
await page.screenshot({ path: 'artifacts/fleet.png', fullPage: true })
await page.goto('http://127.0.0.1:5173/vehicles/LAURORA0000000001', { waitUntil: 'networkidle' })
await page.waitForTimeout(1500)
await page.screenshot({ path: 'artifacts/vehicle-detail.png', fullPage: true })
console.log(JSON.stringify({ screenshots: 4, pageErrors: errors }))
await browser.close()
if (errors.length) process.exitCode = 1

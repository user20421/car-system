import { chromium } from '@playwright/test'
import { mkdir } from 'node:fs/promises'

const browser = await chromium.launch({ headless: true, executablePath: 'C:/Program Files/Google/Chrome/Application/chrome.exe' })
const page = await browser.newPage({ viewport: { width: 430, height: 932 }, deviceScaleFactor: 1 })
const errors = []
page.on('pageerror', error => errors.push(error.message))
await mkdir('artifacts', { recursive: true })
await page.goto('http://127.0.0.1:5000', { waitUntil: 'networkidle' })
await page.waitForTimeout(4000)
await page.screenshot({ path: 'artifacts/flutter-login.png', fullPage: true })
await page.mouse.click(215, 606)
await page.waitForTimeout(5000)
await page.screenshot({ path: 'artifacts/flutter-garage.png', fullPage: true })
await page.setViewportSize({ width: 1280, height: 900 })
await page.waitForTimeout(1500)
await page.screenshot({ path: 'artifacts/flutter-wide.png', fullPage: true })
console.log(JSON.stringify({ screenshots: 3, pageErrors: errors }))
await browser.close()
if (errors.length) process.exitCode = 1

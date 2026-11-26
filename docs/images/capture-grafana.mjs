import { chromium } from "playwright";

const boards = [
  ["rackpilot-gpu-dashboard.png", "/d/rackpilot-gpu-synthetic/rackpilot-gpu-synthetic?orgId=1"],
  ["rackpilot-bmc-dashboard.png", "/d/rackpilot-bmc-synthetic/rackpilot-bmc-synthetic?orgId=1"],
  ["rackpilot-kubernetes-dashboard.png", "/d/rackpilot-kubernetes-overview/rackpilot-kubernetes-overview?orgId=1"],
];

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
await page.goto("http://127.0.0.1:3000/login");
await page.locator('input[name="user"]').fill("admin");
await page.locator('input[name="password"]').fill("rackpilot-dev");
await page.locator('button[type="submit"]').click();
await page.waitForTimeout(3000);
for (const [name, path] of boards) {
  await page.goto(`http://127.0.0.1:3000${path}`, { waitUntil: "networkidle", timeout: 120000 });
  await page.waitForTimeout(8000);
  await page.screenshot({ path: name, fullPage: true });
}
await browser.close();
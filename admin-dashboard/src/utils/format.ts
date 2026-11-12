export const formatNumber = (value: number, digits = 0) => new Intl.NumberFormat('zh-CN', { maximumFractionDigits: digits, minimumFractionDigits: digits }).format(value)

export function formatTime(timestamp?: number | string | null, withDate = false) {
  if (!timestamp) return '—'
  const date = new Date(timestamp)
  if (Number.isNaN(date.getTime())) return '—'
  return new Intl.DateTimeFormat('zh-CN', {
    ...(withDate ? { month: '2-digit', day: '2-digit' } : {}),
    hour: '2-digit', minute: '2-digit', second: withDate ? undefined : '2-digit', hour12: false,
  }).format(date)
}

export function relativeTime(timestamp?: number) {
  if (!timestamp) return '暂无数据'
  const seconds = Math.max(0, Math.round((Date.now() - timestamp) / 1000))
  if (seconds < 10) return '刚刚'
  if (seconds < 60) return `${seconds} 秒前`
  if (seconds < 3600) return `${Math.floor(seconds / 60)} 分钟前`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} 小时前`
  return `${Math.floor(seconds / 86400)} 天前`
}

export function shortVin(vin: string) { return `${vin.slice(0, 4)}••••${vin.slice(-4)}` }
export function isTerminal(status: string) { return ['succeeded', 'failed', 'expired'].includes(status) }
export function commandLabel(type: string) {
  return ({ lock: '车辆闭锁', unlock: '车辆解锁', hvac_on: '开启空调', hvac_off: '关闭空调', set_temperature: '设定温度', defrost_on: '开启除霜', defrost_off: '关闭除霜', honk_flash: '鸣笛闪灯' } as Record<string, string>)[type] || type
}
export function statusLabel(status: string) {
  return ({ pending: '等待下发', sent: '已发送', received: '车辆已接收', succeeded: '执行成功', failed: '执行失败', expired: '已过期', open: '待处理', acknowledged: '已确认', online: '在线', offline: '离线' } as Record<string, string>)[status] || status
}

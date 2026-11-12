import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import MetricCard from '../components/common/MetricCard.vue'
import StatusBadge from '../components/common/StatusBadge.vue'
import VehicleArt from '../components/vehicle/VehicleArt.vue'

describe('core visual components', () => {
  it('renders metric values, units and trend semantics', () => {
    const wrapper = mount(MetricCard, { props: { label: '当前在线', value: 24, unit: '辆', trend: 2.4, points: [1, 2, 3] } })
    expect(wrapper.text()).toContain('当前在线')
    expect(wrapper.text()).toContain('24')
    expect(wrapper.text()).toContain('2.4%')
    expect(wrapper.find('svg.spark').exists()).toBe(true)
  })
  it('maps command and presence states to readable badges', () => {
    expect(mount(StatusBadge, { props: { status: 'received' } }).text()).toContain('车辆已接收')
    expect(mount(StatusBadge, { props: { status: 'failed' } }).classes()).toContain('badge-danger')
    expect(mount(StatusBadge, { props: { status: 'online', pulse: true } }).find('.dot').exists()).toBe(true)
  })
  it('ships an original accessible vehicle illustration', () => {
    const wrapper = mount(VehicleArt)
    expect(wrapper.find('svg').attributes('aria-label')).toContain('AURORA')
    expect(wrapper.findAll('circle').length).toBeGreaterThan(4)
  })
})

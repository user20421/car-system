import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { api } from '../api'
import { CommandTracker } from '../services/commands'
import type { CommandType, VehicleCommand } from '../types/api'

export const useCommandStore = defineStore('commands', () => {
  const tracker = new CommandTracker(api)
  const commands = ref<Record<string, VehicleCommand>>({})
  const sending = ref<Record<string, boolean>>({})
  const errors = ref<Record<string, string>>({})
  const history = computed(() => Object.values(commands.value).sort((a, b) => b.created_at_ms - a.created_at_ms))

  async function send(vin: string, type: CommandType, params: Record<string, string> = {}) {
    const key = `${vin}:${type}`
    if (sending.value[key]) throw new Error('相同指令正在执行，请勿重复提交')
    sending.value[key] = true
    errors.value[key] = ''
    try {
      return await tracker.send(vin, type, params, 60_000, command => {
        commands.value[command.command_id] = command
      })
    } catch (error) {
      errors.value[key] = error instanceof Error ? error.message : '指令发送失败'
      throw error
    } finally { sending.value[key] = false }
  }

  async function track(command: VehicleCommand) {
    commands.value[command.command_id] = command
    const result = await tracker.poll(command, update => { commands.value[update.command_id] = update })
    commands.value[result.command_id] = result
    return result
  }

  return { commands, history, sending, errors, send, track, pendingCount: computed(() => tracker.pendingCount) }
})

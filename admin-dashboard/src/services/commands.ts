import type { ApiAdapter, CommandStatus, CommandType, CreateCommandInput, VehicleCommand } from '../types/api'
import { isTerminal } from '../utils/format'

export const terminalStatuses: CommandStatus[] = ['succeeded', 'failed', 'expired']

export class CommandTracker {
  private readonly active = new Map<string, Promise<VehicleCommand>>()
  private readonly timers = new Set<ReturnType<typeof setTimeout>>()

  constructor(private readonly adapter: Pick<ApiAdapter, 'createCommand' | 'getCommand'>) {}

  send(vin: string, type: CommandType, params: Record<string, string> = {}, ttlMs = 60_000, onUpdate?: (command: VehicleCommand) => void) {
    const operationKey = `${vin}:${type}:${JSON.stringify(params)}`
    const existing = this.active.get(operationKey)
    if (existing) return existing
    const input: CreateCommandInput = {
      type, params, idempotency_key: crypto.randomUUID(), expires_at_ms: Date.now() + ttlMs,
    }
    const promise = this.adapter.createCommand(vin, input)
      .then(command => this.poll(command, onUpdate))
      .finally(() => this.active.delete(operationKey))
    this.active.set(operationKey, promise)
    return promise
  }

  async poll(initial: VehicleCommand, onUpdate?: (command: VehicleCommand) => void): Promise<VehicleCommand> {
    let command = initial
    onUpdate?.(command)
    while (!isTerminal(command.status) && Date.now() < command.expires_at_ms) {
      await new Promise<void>(resolve => {
        const timer = setTimeout(resolve, 700)
        this.timers.add(timer)
      })
      command = await this.adapter.getCommand(command.vin, command.command_id)
      onUpdate?.(command)
    }
    return command
  }

  get pendingCount() { return this.active.size }
  dispose() { this.timers.forEach(clearTimeout); this.timers.clear(); this.active.clear() }
}

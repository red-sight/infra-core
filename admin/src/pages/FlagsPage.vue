<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Switch from '@/components/ui/Switch.vue'
import SearchBox from '@/components/ui/SearchBox.vue'
import Segmented from '@/components/ui/Segmented.vue'
import IconChip from '@/components/ui/IconChip.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import Avatar from '@/components/ui/Avatar.vue'
import { useToast } from '@/composables/useToast'
import { FLAGS, type Flag } from '@/lib/data'
import type { BadgeVariant } from '@/lib/types'

const toast = useToast()
const flags = ref<Flag[]>(FLAGS.map((f) => ({ ...f })))
const q = ref('')
const view = ref('all')

const ENV_VARIANT: Record<string, BadgeVariant> = { all: 'success', prod: 'info', staging: 'warning' }
const KIND_VARIANT: Record<string, BadgeVariant> = { release: 'muted', experiment: 'violet', ops: 'outline' }

const filtered = computed(() =>
  flags.value.filter(
    (f) =>
      (view.value === 'all' || (view.value === 'on' && f.enabled) || (view.value === 'off' && !f.enabled)) &&
      (!q.value || f.name.toLowerCase().includes(q.value.toLowerCase()) || f.id.includes(q.value.toLowerCase())),
  ),
)
const onCount = computed(() => flags.value.filter((f) => f.enabled).length)

function toggle(f: Flag) {
  flags.value = flags.value.map((x) =>
    x.id === f.id ? { ...x, enabled: !x.enabled, rollout: !x.enabled ? x.rollout || 100 : x.rollout } : x,
  )
  toast({ title: f.enabled ? 'Flag disabled' : 'Flag enabled', desc: f.name, variant: f.enabled ? 'error' : 'success' })
}
function setRollout(f: Flag, v: number) {
  flags.value = flags.value.map((x) => (x.id === f.id ? { ...x, rollout: v } : x))
}
function flagIcon(kind: string) {
  return kind === 'experiment' ? 'zap' : kind === 'ops' ? 'settings' : 'flag'
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Feature Flags</h1>
        <p class="page-sub">
          {{ onCount }} of {{ flags.length }} features enabled. Roll features out gradually and target specific environments.
        </p>
      </div>
      <div class="page-actions">
        <Button variant="primary" icon="plus" @click="toast({ title: 'New flag', desc: 'Flag editor would open here', variant: 'info' })">New flag</Button>
      </div>
    </div>

    <Card>
      <div style="padding: 14px; display: flex; gap: 12px; align-items: center; flex-wrap: wrap; border-bottom: 1px solid var(--border)">
        <div style="flex: 1 1 220px; min-width: 180px; display: flex"><SearchBox v-model="q" placeholder="Search flags…" /></div>
        <Segmented
          v-model="view"
          :options="[{ value: 'all', label: 'All' }, { value: 'on', label: 'Enabled' }, { value: 'off', label: 'Disabled' }]"
        />
      </div>

      <EmptyState v-if="filtered.length === 0" icon="flag" title="No flags found" desc="Try a different search or filter." />

      <div v-else>
        <div
          v-for="(f, i) in filtered"
          :key="f.id"
          :style="{ display: 'flex', alignItems: 'center', gap: '16px', padding: '16px 20px', borderTop: i ? '1px solid var(--border)' : 'none' }"
        >
          <IconChip :name="flagIcon(f.kind)" :tone="f.enabled ? 'success' : 'muted'" :size="38" :icon-size="17" />
          <div style="flex: 1; min-width: 0">
            <div style="display: flex; align-items: center; gap: 9px; flex-wrap: wrap">
              <span style="font-weight: 600">{{ f.name }}</span>
              <span class="mono" style="font-size: 11.5px; color: var(--muted-foreground); background: var(--muted); padding: 1px 7px; border-radius: 5px">{{ f.id }}</span>
              <Badge :variant="KIND_VARIANT[f.kind]">{{ f.kind }}</Badge>
            </div>
            <div style="font-size: 12.5px; color: var(--muted-foreground); margin-top: 3px">{{ f.desc }}</div>
            <div style="display: flex; gap: 14px; margin-top: 8px; font-size: 12px; color: var(--muted-foreground); flex-wrap: wrap; align-items: center">
              <Badge :variant="ENV_VARIANT[f.env]" dot>{{ f.env === 'all' ? 'All envs' : f.env }}</Badge>
              <span class="dotline"><Icon name="clock" :size="13" /> {{ f.updated }}</span>
              <span class="dotline"><Avatar :name="f.owner" size="xs" /> {{ f.owner }}</span>
            </div>
          </div>

          <div style="width: 160px; flex-shrink: 0" class="flag-rollout">
            <div style="display: flex; justify-content: space-between; font-size: 11.5px; color: var(--muted-foreground); margin-bottom: 5px">
              <span>Rollout</span>
              <span class="tnum" :style="{ fontWeight: 600, color: f.enabled ? 'var(--foreground)' : 'var(--muted-foreground)' }">{{ f.enabled ? f.rollout + '%' : '—' }}</span>
            </div>
            <input
              type="range"
              min="0"
              max="100"
              step="5"
              :value="f.rollout"
              :disabled="!f.enabled"
              class="range"
              style="width: 100%"
              @input="setRollout(f, Number(($event.target as HTMLInputElement).value))"
            />
          </div>

          <Switch :checked="f.enabled" @change="toggle(f)" />
        </div>
      </div>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Card from '@/components/ui/Card.vue'
import CardHead from '@/components/ui/CardHead.vue'
import CardBody from '@/components/ui/CardBody.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Avatar from '@/components/ui/Avatar.vue'
import IconChip from '@/components/ui/IconChip.vue'
import AreaLine from '@/components/charts/AreaLine.vue'
import Donut from '@/components/charts/Donut.vue'
import { useNav } from '@/composables/useNav'
import { AUDIT, AUDIT_CAT, KPIS, ORGS, PLAN_MIX, STATUS, TREND, fmtMoney } from '@/lib/data'

const nav = useNav()

const attention = computed(() =>
  ORGS.filter((o) => ['past_due', 'suspended', 'trial'].includes(o.status)).slice(0, 4),
)
const recent = computed(() => AUDIT.slice(0, 6))
const planTotal = computed(() => PLAN_MIX.reduce((s, x) => s + x.count, 0))

const trendKeys = [
  { key: 'signups', label: 'Signups', color: 'var(--chart-1)' },
  { key: 'churn', label: 'Churn', color: 'var(--chart-5)' },
]

function activityIcon(cat: string) {
  return cat === 'billing' ? 'credit-card' : cat === 'security' ? 'shield' : cat === 'platform' ? 'flag' : 'building-2'
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Overview</h1>
        <p class="page-sub">Platform health across all tenant organizations. Showing data for the last 30 days.</p>
      </div>
      <div class="page-actions">
        <Button variant="outline" icon="download">Export</Button>
        <Button variant="primary" icon="plus" @click="nav('organizations', { create: '1' })">New organization</Button>
      </div>
    </div>

    <div class="section-stack">
      <div class="grid-kpi">
        <Card v-for="k in KPIS" :key="k.id" hover class="kpi">
          <div class="kpi__top">
            <span class="kpi__label">{{ k.label }}</span>
            <IconChip :name="k.icon" :tone="k.tone" :size="32" :icon-size="16" />
          </div>
          <div class="kpi__value tnum">{{ k.value }}</div>
          <div class="kpi__foot">
            <span :class="['delta', 'delta--' + k.dir]">
              <Icon :name="k.dir === 'up' ? 'trending-up' : 'trending-down'" />{{ k.delta }}
            </span>
            <span style="color: var(--muted-foreground)">{{ k.sub }}</span>
          </div>
        </Card>
      </div>

      <div class="grid-2">
        <Card>
          <CardHead bordered title="Signups vs. churn" desc="Net organization growth, last 12 weeks">
            <template #actions>
              <div style="display: flex; gap: 14px; font-size: 12px">
                <span class="dotline"><span style="width: 9px; height: 9px; border-radius: 3px; background: var(--chart-1)" /> Signups</span>
                <span class="dotline"><span style="width: 9px; height: 9px; border-radius: 3px; background: var(--chart-5)" /> Churn</span>
              </div>
            </template>
          </CardHead>
          <CardBody>
            <AreaLine :data="TREND" :height="220" :keys="trendKeys" />
          </CardBody>
        </Card>

        <Card>
          <CardHead bordered title="Plan distribution" desc="Active organizations by plan" />
          <CardBody>
            <div style="display: flex; flex-direction: column; align-items: center; gap: 18px">
              <Donut :data="PLAN_MIX" />
              <div style="width: 100%; display: flex; flex-direction: column; gap: 9px">
                <div
                  v-for="p in PLAN_MIX"
                  :key="p.plan"
                  style="display: flex; align-items: center; gap: 10px; font-size: 13px"
                >
                  <span :style="{ width: '9px', height: '9px', borderRadius: '3px', background: p.color }" />
                  <span style="flex: 1; text-transform: capitalize">{{ p.plan }}</span>
                  <span class="tnum" style="font-weight: 600">{{ p.count }}</span>
                  <span class="cell-muted tnum" style="width: 42px; text-align: right">
                    {{ Math.round((p.count / planTotal) * 100) }}%
                  </span>
                </div>
              </div>
            </div>
          </CardBody>
        </Card>
      </div>

      <div class="grid-2">
        <Card>
          <CardHead bordered title="Needs attention" desc="Organizations with billing or access issues">
            <template #actions>
              <Button variant="ghost" size="sm" icon-right="arrow-right" :icon-size="14" @click="nav('organizations')">View all</Button>
            </template>
          </CardHead>
          <div>
            <button
              v-for="(o, i) in attention"
              :key="o.id"
              class="attention-row"
              :style="{ borderTop: i ? '1px solid var(--border)' : 'none' }"
              @click="nav('organizations', { open: o.id })"
            >
              <Avatar :name="o.name" size="sm" square />
              <div style="flex: 1; min-width: 0">
                <div style="font-weight: 600">{{ o.name }}</div>
                <div style="font-size: 12px; color: var(--muted-foreground)">
                  {{ fmtMoney(o.mrr) }}/mo · {{ o.seatsUsed }}/{{ o.seats }} seats
                </div>
              </div>
              <Badge :variant="STATUS[o.status].variant" dot>{{ STATUS[o.status].label }}</Badge>
              <Icon name="chevron-right" :size="15" style="color: var(--muted-foreground)" />
            </button>
          </div>
        </Card>

        <Card>
          <CardHead bordered title="Recent activity">
            <template #actions>
              <Button variant="ghost" size="sm" icon-right="arrow-right" :icon-size="14" @click="nav('audit')">Audit log</Button>
            </template>
          </CardHead>
          <div style="padding: 0">
            <div
              v-for="(a, i) in recent"
              :key="a.id"
              :style="{ display: 'flex', gap: '11px', padding: '12px 24px', borderTop: i ? '1px solid var(--border)' : 'none' }"
            >
              <IconChip :name="activityIcon(a.cat)" :tone="AUDIT_CAT[a.cat]" :size="30" :icon-size="14" />
              <div style="flex: 1; min-width: 0">
                <div style="font-size: 13px">
                  <b>{{ a.actor }}</b>
                  <span class="mono" style="font-size: 12px; color: var(--muted-foreground)">{{ a.action }}</span>
                </div>
                <div style="font-size: 12px; color: var(--muted-foreground); white-space: nowrap; overflow: hidden; text-overflow: ellipsis">
                  {{ a.target }}
                </div>
              </div>
              <span style="font-size: 12px; color: var(--muted-foreground); white-space: nowrap">{{ a.ago }}</span>
            </div>
          </div>
        </Card>
      </div>
    </div>
  </div>
</template>

<style scoped>
.attention-row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 13px 24px;
  text-align: left;
  transition: background 0.12s ease;
}
.attention-row:hover {
  background: var(--subtle);
}
</style>

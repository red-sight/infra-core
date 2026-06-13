<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Card from '@/components/ui/Card.vue'
import CardHead from '@/components/ui/CardHead.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Avatar from '@/components/ui/Avatar.vue'
import SearchBox from '@/components/ui/SearchBox.vue'
import Select from '@/components/ui/Select.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import MenuItem from '@/components/ui/MenuItem.vue'
import MenuSep from '@/components/ui/MenuSep.vue'
import IconChip from '@/components/ui/IconChip.vue'
import { useToast } from '@/composables/useToast'
import { INVOICES, ORGS, PLANS, PLAN_PRICING, fmtMoney } from '@/lib/data'

const toast = useToast()
const q = ref('')
const status = ref('')

const totalMrr = ORGS.reduce((s, o) => s + o.mrr, 0)
const pastDue = INVOICES.filter((i) => i.status === 'past_due')

const kpis = [
  { label: 'MRR', value: fmtMoney(totalMrr), icon: 'credit-card', tone: 'success', delta: '+8.4%', dir: 'up' },
  { label: 'ARR (run-rate)', value: fmtMoney(totalMrr * 12), icon: 'trending-up', tone: 'primary', delta: '+11.2%', dir: 'up' },
  { label: 'Past due', value: fmtMoney(pastDue.reduce((s, i) => s + i.amount, 0)), icon: 'alert-triangle', tone: 'danger', delta: pastDue.length + ' invoices', dir: 'down' },
  { label: 'Paying orgs', value: String(ORGS.filter((o) => o.mrr > 0).length), icon: 'building-2', tone: 'info', delta: '+3', dir: 'up' },
]

const invoices = computed(() =>
  INVOICES.filter(
    (i) =>
      (!q.value || i.org.toLowerCase().includes(q.value.toLowerCase()) || i.id.toLowerCase().includes(q.value.toLowerCase())) &&
      (!status.value || i.status === status.value),
  ),
)

function planCount(p: string) {
  return ORGS.filter((o) => o.plan === p).length
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Billing &amp; Subscriptions</h1>
        <p class="page-sub">Revenue, plans and invoices across all organizations.</p>
      </div>
      <div class="page-actions">
        <Button variant="outline" icon="download">Export</Button>
        <Button variant="outline" icon="settings">Plan settings</Button>
      </div>
    </div>

    <div class="section-stack">
      <div class="grid-kpi">
        <Card v-for="k in kpis" :key="k.label" pad class="kpi">
          <div class="kpi__top"><span class="kpi__label">{{ k.label }}</span><IconChip :name="k.icon" :tone="k.tone" :size="32" :icon-size="16" /></div>
          <div class="kpi__value tnum">{{ k.value }}</div>
          <div class="kpi__foot">
            <span :class="['delta', 'delta--' + k.dir]"><Icon :name="k.dir === 'up' ? 'trending-up' : 'alert-circle'" />{{ k.delta }}</span>
          </div>
        </Card>
      </div>

      <div>
        <div style="font-weight: 600; font-size: 15px; margin-bottom: 14px">Plans</div>
        <div class="grid-cards">
          <Card v-for="p in PLAN_PRICING" :key="p.plan" pad hover style="display: flex; flex-direction: column; gap: 14px">
            <div style="display: flex; justify-content: space-between; align-items: flex-start">
              <div>
                <Badge :variant="PLANS[p.plan].variant">{{ PLANS[p.plan].label }}</Badge>
                <div style="margin-top: 10px; font-size: 26px; font-weight: 700; letter-spacing: -0.03em" class="tnum">
                  {{ p.price === null ? 'Custom' : p.price === 0 ? '$0' : `$${p.price}` }}
                  <span v-if="p.price" style="font-size: 12.5px; color: var(--muted-foreground); font-weight: 500"> /seat/mo</span>
                </div>
              </div>
              <div style="text-align: right">
                <div style="font-size: 22px; font-weight: 700" class="tnum">{{ planCount(p.plan) }}</div>
                <div style="font-size: 11.5px; color: var(--muted-foreground)">orgs</div>
              </div>
            </div>
            <div style="display: flex; flex-direction: column; gap: 7px; font-size: 12.5px; color: var(--muted-foreground)">
              <span v-for="feat in p.features.slice(0, 4)" :key="feat" class="dotline">
                <Icon name="check" :size="13" style="color: var(--success)" /> {{ feat }}
              </span>
            </div>
          </Card>
        </div>
      </div>

      <Card>
        <CardHead bordered title="Recent invoices" desc="Latest billing activity">
          <template #actions>
            <div style="display: flex; gap: 9px">
              <div style="width: 200px; display: flex"><SearchBox v-model="q" placeholder="Search invoices…" /></div>
              <Select
                v-model="status"
                :options="[{ value: '', label: 'All' }, { value: 'active', label: 'Paid' }, { value: 'past_due', label: 'Past due' }]"
                filter
              />
            </div>
          </template>
        </CardHead>
        <div class="table-wrap">
          <table class="table">
            <thead>
              <tr><th>Invoice</th><th>Organization</th><th>Plan</th><th class="th-right">Amount</th><th>Method</th><th>Date</th><th>Status</th><th style="width: 44px"></th></tr>
            </thead>
            <tbody>
              <tr v-for="inv in invoices" :key="inv.id">
                <td class="mono cell-strong">{{ inv.id }}</td>
                <td><div class="idcell"><Avatar :name="inv.org" size="xs" square /><span class="idcell__name">{{ inv.org }}</span></div></td>
                <td><Badge :variant="PLANS[inv.plan].variant">{{ PLANS[inv.plan].label }}</Badge></td>
                <td class="td-right cell-strong tnum">{{ fmtMoney(inv.amount) }}</td>
                <td><span class="cell-muted mono" style="font-size: 12.5px">{{ inv.method }}</span></td>
                <td><span class="cell-muted tnum">{{ inv.date }}</span></td>
                <td>
                  <Badge v-if="inv.status === 'past_due'" variant="danger" dot>Past due</Badge>
                  <Badge v-else variant="success" dot>Paid</Badge>
                </td>
                <td>
                  <Dropdown align="end">
                    <template #trigger>
                      <button class="btn btn--ghost btn--icon btn--sm row-actions"><Icon name="more-horizontal" :size="16" /></button>
                    </template>
                    <MenuItem icon="eye">View invoice</MenuItem>
                    <MenuItem icon="download" @click="toast({ title: 'Invoice downloaded', desc: inv.id })">Download PDF</MenuItem>
                    <MenuItem v-if="inv.status === 'past_due'" icon="send" @click="toast({ title: 'Reminder sent', desc: inv.org, variant: 'info' })">Send reminder</MenuItem>
                    <MenuSep />
                    <MenuItem icon="rotate-ccw" danger @click="toast({ title: 'Refund issued', desc: `${inv.id} · ${fmtMoney(inv.amount)}`, variant: 'error' })">Issue refund</MenuItem>
                  </Dropdown>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  </div>
</template>

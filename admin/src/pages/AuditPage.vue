<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Avatar from '@/components/ui/Avatar.vue'
import SearchBox from '@/components/ui/SearchBox.vue'
import Segmented from '@/components/ui/Segmented.vue'
import IconChip from '@/components/ui/IconChip.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { useToast } from '@/composables/useToast'
import { AUDIT, AUDIT_CAT } from '@/lib/data'

const toast = useToast()
const q = ref('')
const cat = ref('all')

const CAT_LABEL: Record<string, string> = { orgs: 'Organizations', billing: 'Billing', security: 'Security', platform: 'Platform' }
const CAT_ICON: Record<string, string> = { orgs: 'building-2', billing: 'credit-card', security: 'shield', platform: 'flag' }

const filtered = computed(() =>
  AUDIT.filter(
    (a) =>
      (cat.value === 'all' || a.cat === cat.value) &&
      (!q.value ||
        a.actor.toLowerCase().includes(q.value.toLowerCase()) ||
        a.action.toLowerCase().includes(q.value.toLowerCase()) ||
        a.target.toLowerCase().includes(q.value.toLowerCase()) ||
        a.org.toLowerCase().includes(q.value.toLowerCase())),
  ),
)
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1 class="page-title">Audit Log</h1>
        <p class="page-sub">An immutable record of every administrative action. Retained for 2 years.</p>
      </div>
      <div class="page-actions">
        <Button variant="outline" icon="download" @click="toast({ title: 'Export started', desc: 'Audit log · CSV', variant: 'info' })">Export log</Button>
      </div>
    </div>

    <Card>
      <div style="padding: 14px; display: flex; gap: 12px; align-items: center; flex-wrap: wrap; border-bottom: 1px solid var(--border)">
        <div style="flex: 1 1 220px; min-width: 180px; display: flex"><SearchBox v-model="q" placeholder="Search actor, action, target…" /></div>
        <Segmented
          v-model="cat"
          :options="[
            { value: 'all', label: 'All' },
            { value: 'orgs', label: 'Orgs' },
            { value: 'billing', label: 'Billing' },
            { value: 'security', label: 'Security' },
            { value: 'platform', label: 'Platform' },
          ]"
        />
      </div>

      <EmptyState v-if="filtered.length === 0" icon="scroll-text" title="No matching events" desc="Try a different search or category." />

      <div v-else class="table-wrap">
        <table class="table">
          <thead>
            <tr><th>Event</th><th>Actor</th><th>Organization</th><th>Category</th><th>IP address</th><th class="th-right">Time</th></tr>
          </thead>
          <tbody>
            <tr v-for="a in filtered" :key="a.id">
              <td>
                <div class="idcell">
                  <IconChip :name="CAT_ICON[a.cat]" :tone="AUDIT_CAT[a.cat]" :size="32" :icon-size="15" />
                  <div class="idcell__main">
                    <div class="mono" style="font-weight: 600; font-size: 12.5px">{{ a.action }}</div>
                    <div class="idcell__sub" style="max-width: 320px">{{ a.target }}</div>
                  </div>
                </div>
              </td>
              <td>
                <span v-if="a.actor === 'System'" class="dotline cell-muted"><Icon name="cpu" :size="14" /> System</span>
                <div v-else class="idcell"><Avatar :name="a.actor" size="xs" /><span>{{ a.actor }}</span></div>
              </td>
              <td><span class="cell-muted">{{ a.org }}</span></td>
              <td><Badge :variant="AUDIT_CAT[a.cat]">{{ CAT_LABEL[a.cat] }}</Badge></td>
              <td><span class="mono cell-muted" style="font-size: 12.5px">{{ a.ip }}</span></td>
              <td class="td-right"><span class="cell-muted tnum" :title="a.time">{{ a.ago }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
      <div style="padding: 12px 16px; border-top: 1px solid var(--border); font-size: 12.5px; color: var(--muted-foreground)">
        Showing {{ filtered.length }} of {{ AUDIT.length }} events
      </div>
    </Card>
  </div>
</template>

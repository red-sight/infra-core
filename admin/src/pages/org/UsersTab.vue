<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { refDebounced } from '@vueuse/core'
import Card from '@/components/ui/Card.vue'
import Icon from '@/components/ui/Icon.vue'
import Avatar from '@/components/ui/Avatar.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import SearchBox from '@/components/ui/SearchBox.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { ApiError } from '@/lib/api'
import { useOrgMembers, type MembersParams, type Organization } from '@/composables/useOrganizations'

const props = defineProps<{ org: Organization }>()

const PAGE_SIZE = 20

const q = ref('')
const debouncedQ = refDebounced(q, 300)
const page = ref(1)
watch(debouncedQ, () => (page.value = 1))

const id = computed(() => props.org.id)
const params = computed<MembersParams>(() => ({
  page: page.value,
  page_size: PAGE_SIZE,
  q: debouncedQ.value.trim() || undefined,
}))

const { data, isLoading, isError, error, isFetching, refetch } = useOrgMembers(id, params)

const members = computed(() => data.value?.items ?? [])
const total = computed(() => data.value?.total ?? 0)
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_SIZE)))
const rangeFrom = computed(() => (total.value === 0 ? 0 : (page.value - 1) * PAGE_SIZE + 1))
const rangeTo = computed(() => Math.min(page.value * PAGE_SIZE, total.value))
</script>

<template>
  <!-- The org has no identity-provider presence yet, so it can't have members. -->
  <EmptyState
    v-if="!org.synced"
    icon="users"
    title="No members yet"
    desc="This organization is still being provisioned in the identity provider. Members appear once sync completes."
  />

  <Card v-else>
    <div style="padding: 14px; border-bottom: 1px solid var(--border)">
      <div class="toolbar">
        <div style="flex: 1 1 240px; min-width: 200px; display: flex">
          <SearchBox v-model="q" placeholder="Search by name or email…" />
        </div>
        <span v-if="isFetching && !isLoading" style="font-size: 12px; color: var(--muted-foreground)">Updating…</span>
      </div>
    </div>

    <div v-if="isLoading" style="padding: 48px; text-align: center; color: var(--muted-foreground)">
      <Icon name="refresh-cw" :size="20" /> Loading members…
    </div>

    <EmptyState
      v-else-if="isError"
      icon="alert-triangle"
      title="Couldn’t load members"
      :desc="error instanceof ApiError ? error.message : 'Request failed. Check your connection and try again.'"
    >
      <template #action>
        <Button variant="outline" size="sm" icon="refresh-cw" @click="refetch()">Retry</Button>
      </template>
    </EmptyState>

    <EmptyState
      v-else-if="members.length === 0"
      icon="users"
      :title="debouncedQ ? 'No matching members' : 'No members yet'"
      :desc="debouncedQ ? 'Try a different search term.' : 'No users have been added to this organization.'"
    >
      <template #action>
        <Button v-if="debouncedQ" variant="outline" size="sm" @click="q = ''">Clear search</Button>
      </template>
    </EmptyState>

    <div v-else class="table-wrap">
      <table class="table">
        <thead>
          <tr>
            <th>Member</th>
            <th>Email</th>
            <th>Roles</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in members" :key="m.id">
            <td>
              <div class="idcell">
                <Avatar :name="m.name" size="sm" />
                <div class="idcell__main">
                  <div class="idcell__name">{{ m.name || '—' }}</div>
                  <div class="idcell__sub mono">{{ m.id }}</div>
                </div>
              </div>
            </td>
            <td><span class="cell-muted">{{ m.email || '—' }}</span></td>
            <td>
              <div v-if="m.roles.length" style="display: flex; gap: 6px; flex-wrap: wrap">
                <Badge v-for="r in m.roles" :key="r.id" variant="muted">{{ r.name }}</Badge>
              </div>
              <span v-else class="cell-muted">—</span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="!isLoading && !isError && members.length"
      style="padding: 12px 16px; border-top: 1px solid var(--border); display: flex; justify-content: space-between; align-items: center"
    >
      <span style="font-size: 12.5px; color: var(--muted-foreground)">
        Showing {{ rangeFrom }}–{{ rangeTo }} of {{ total }}
      </span>
      <div style="display: flex; gap: 4px; align-items: center">
        <Button variant="outline" size="sm" icon="chevron-left" :icon-size="14" :disabled="page <= 1" @click="page--">
          Prev
        </Button>
        <span style="font-size: 12.5px; color: var(--muted-foreground); padding: 0 6px">
          Page {{ page }} / {{ totalPages }}
        </span>
        <Button variant="outline" size="sm" icon-right="chevron-right" :icon-size="14" :disabled="page >= totalPages" @click="page++">
          Next
        </Button>
      </div>
    </div>
  </Card>
</template>

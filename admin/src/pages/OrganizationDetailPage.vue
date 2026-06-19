<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Avatar from '@/components/ui/Avatar.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Icon from '@/components/ui/Icon.vue'
import Tabs from '@/components/ui/Tabs.vue'
import Breadcrumb, { type Crumb } from '@/components/ui/Breadcrumb.vue'
import EmptyState from '@/components/ui/EmptyState.vue'
import { useOrganization } from '@/composables/useOrganizations'
import { ApiError } from '@/lib/api'

const route = useRoute()
const router = useRouter()

const id = computed(() => String(route.params.id))
const { data: org, isLoading, isError, error, refetch } = useOrganization(id)

// Each tab is a nested route named `organization-<value>`. More land per slice
// (users, roles, settings); the active tab derives from the current route name.
const tabs = [
  { value: 'overview', label: 'Overview' },
  { value: 'users', label: 'Users' },
]
const activeTab = computed(() => String(route.name ?? '').replace('organization-', '') || 'overview')
function selectTab(value: string) {
  router.push({ name: `organization-${value}`, params: { id: id.value } })
}

const crumbs = computed<Crumb[]>(() => [
  { label: 'Organizations', to: { name: 'organizations' } },
  { label: org.value?.name ?? 'Organization' },
])
</script>

<template>
  <div class="page">
    <Breadcrumb :items="crumbs" />

    <!-- Loading (first load) -->
    <div v-if="isLoading" style="padding: 48px; text-align: center; color: var(--muted-foreground)">
      <Icon name="refresh-cw" :size="20" /> Loading organization…
    </div>

    <!-- Error -->
    <EmptyState
      v-else-if="isError"
      icon="alert-triangle"
      title="Couldn’t load organization"
      :desc="error instanceof ApiError ? error.message : 'Request failed. Check your connection and try again.'"
    >
      <template #action>
        <Button variant="outline" size="sm" icon="refresh-cw" @click="refetch()">Retry</Button>
      </template>
    </EmptyState>

    <template v-else-if="org">
      <div class="org-head">
        <Avatar :name="org.name" size="lg" square />
        <div class="org-head__main">
          <h1 class="page-title org-head__title">
            {{ org.name }}
            <Badge v-if="org.is_master" variant="violet">Master</Badge>
            <Badge v-if="org.synced" variant="success" dot>Synced</Badge>
            <Badge v-else variant="warning" dot>Pending</Badge>
          </h1>
          <p v-if="org.description" class="page-sub org-head__desc">{{ org.description }}</p>
          <p class="org-head__domain mono">
            <Icon name="globe" :size="13" /> {{ org.domain }}
          </p>
        </div>
      </div>

      <Tabs :tabs="tabs" :model-value="activeTab" @update:model-value="selectTab" />

      <div class="org-body">
        <RouterView :org="org" />
      </div>
    </template>
  </div>
</template>

<style scoped>
.org-head {
  display: flex;
  gap: 14px;
  align-items: center;
  margin-bottom: 20px;
}
.org-head__main {
  min-width: 0;
}
.org-head__title {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.org-head__desc {
  margin-top: 4px;
}
.org-head__domain {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
  font-size: 13px;
  color: var(--muted-foreground);
}
.org-body {
  margin-top: 22px;
}
</style>

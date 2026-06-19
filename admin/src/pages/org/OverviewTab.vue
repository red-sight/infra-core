<script setup lang="ts">
import Card from '@/components/ui/Card.vue'
import Icon from '@/components/ui/Icon.vue'
import type { Organization } from '@/composables/useOrganizations'

defineProps<{ org: Organization }>()

function fmtDate(s: string) {
  return new Date(s).toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div style="display: flex; flex-direction: column; gap: 18px">
    <Card pad>
      <div style="font-weight: 600; margin-bottom: 14px; font-size: 13.5px">Details</div>
      <dl class="dl">
        <dt>Name</dt><dd>{{ org.name }}</dd>
        <dt>Slug</dt><dd class="mono">{{ org.slug || 'apex domain (master)' }}</dd>
        <dt>Domain</dt><dd class="mono">{{ org.domain }}</dd>
        <dt>Description</dt><dd>{{ org.description || '—' }}</dd>
        <dt>Organization ID</dt><dd class="mono">{{ org.id }}</dd>
        <dt>Logto ID</dt><dd class="mono">{{ org.external_id || '— (not yet provisioned)' }}</dd>
        <dt>Created</dt><dd>{{ fmtDate(org.created_at) }}</dd>
        <dt>Updated</dt><dd>{{ fmtDate(org.updated_at) }}</dd>
      </dl>
    </Card>

    <div
      v-if="!org.synced"
      style="display: flex; gap: 9px; align-items: flex-start; font-size: 12.5px; background: var(--info-bg); color: var(--info-fg); padding: 11px 13px; border-radius: 8px; border: 1px solid var(--info-bd)"
    >
      <Icon name="info" :size="15" style="flex-shrink: 0; margin-top: 1px" />
      This organization is being provisioned in the identity provider. The Logto ID appears once sync completes.
    </div>
  </div>
</template>

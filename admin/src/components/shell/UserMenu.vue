<script setup lang="ts">
import { computed } from 'vue'
import { useAuth } from '@/composables/useAuth'
import { useQuery } from '@tanstack/vue-query'
import Icon from '@/components/ui/Icon.vue'
import Avatar from '@/components/ui/Avatar.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import MenuItem from '@/components/ui/MenuItem.vue'
import MenuSep from '@/components/ui/MenuSep.vue'
import { useNav } from '@/composables/useNav'

withDefaults(defineProps<{ width?: number; align?: 'start' | 'end'; up?: boolean }>(), {
  width: 216,
  align: 'start',
  up: false,
})

const { signOut, fetchUserInfo, isAuthenticated } = useAuth()
const nav = useNav()

const { data: user } = useQuery({
  queryKey: ['user-info'],
  queryFn: () => fetchUserInfo(),
  staleTime: 5 * 60 * 1000,
  enabled: isAuthenticated,
})

const name = computed(() => user.value?.name ?? user.value?.email ?? 'Operator')
const email = computed(() => user.value?.email ?? '—')

function handleSignOut() {
  signOut()
}
</script>

<template>
  <Dropdown :align="align" :width="width" :up="up" block>
    <template #trigger>
      <button class="user-chip user-chip--block">
        <Avatar :name="name" size="sm" />
        <span style="flex: 1; min-width: 0; text-align: left">
          <span class="user-chip__name" style="display: block">{{ name }}</span>
          <span
            style="display: block; font-size: 11.5px; color: var(--muted-foreground); white-space: nowrap; overflow: hidden; text-overflow: ellipsis"
            >{{ email }}</span
          >
        </span>
        <Icon name="chevrons-up-down" :size="15" style="color: var(--muted-foreground); flex-shrink: 0" />
      </button>
    </template>

    <div style="padding: 6px 9px 8px">
      <div style="font-weight: 600; font-size: 13px">{{ name }}</div>
      <div style="font-size: 12px; color: var(--muted-foreground)">{{ email }}</div>
    </div>
    <MenuSep />
    <MenuItem icon="user">Profile</MenuItem>
    <MenuItem icon="settings" @click="nav('flags')">Console settings</MenuItem>
    <MenuItem icon="life-buoy">Support</MenuItem>
    <MenuSep />
    <MenuItem icon="log-out" danger @click="handleSignOut">Sign out</MenuItem>
  </Dropdown>
</template>

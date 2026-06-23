<template>
  <span class="badge" :class="statusBadgeClass(value)">{{ statusValueLabel(value) }}</span>
</template>

<script setup lang="ts">
import { statusBadgeClass, statusValueLabel } from './presentation'

defineProps<{
  value: string
}>()
</script>

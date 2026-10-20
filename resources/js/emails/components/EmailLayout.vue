<script setup lang="ts">
defineProps<{ preheader: string }>();
</script>

<template>
  <Layout body-class="bg-gray-50">
    <Preheader>{{ preheader }}</Preheader>

    <Container class="max-w-xl py-10">
      <table role="none" width="100%" cellpadding="0" cellspacing="0">
        <tr>
          <td class="pb-8 text-center">
            <Text class="mt-0 text-lg font-semibold text-gray-950">KinStack</Text>
          </td>
        </tr>
        <tr>
          <td class="rounded-lg bg-white px-8 py-10 shadow-sm">
            <slot />
          </td>
        </tr>
        <tr>
          <td class="pt-8 text-center">
            <Text class="mt-0 text-xs text-gray-400">
              You are receiving this email because you have a KinStack account.
            </Text>
          </td>
        </tr>
      </table>
    </Container>
  </Layout>
</template>

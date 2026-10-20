<template>
  <EmailLayout preheader="Your sign-in link for your KinStack account.">
    <Heading :level="2" class="text-xl font-semibold text-gray-950"> Sign in to KinStack </Heading>

    <Text> We received a request to sign in to your account without a password. </Text>

    <Button href="[[.Link]]" align="center" class="mt-6"> Sign in now </Button>

    <Text class="mt-6 text-sm text-gray-500">
      This link expires on [[.ExpiresAt]] and can only be used once. If you did not request it, you can safely ignore
      this email.
    </Text>
  </EmailLayout>
</template>

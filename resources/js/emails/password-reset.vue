<template>
  <EmailLayout preheader="Reset the password for your KinStack account.">
    <Heading :level="2" class="text-xl font-semibold text-gray-950"> Reset password </Heading>

    <Text> We received a request to reset the password for your KinStack account. </Text>

    <Button href="[[.Link]]" align="center" class="mt-6"> Reset password </Button>

    <Text class="mt-6 text-sm text-gray-500">
      This link expires on [[.ExpiresAt]] and can only be used once. If you did not request it, you can safely ignore
      this email.
    </Text>
  </EmailLayout>
</template>

<template>
  <EmailLayout preheader="Verify the email address for your KinStack account.">
    <Heading :level="2" class="text-xl font-semibold text-gray-950"> Welcome to KinStack </Heading>

    <Text> Confirm this email address to finish setting up your account. </Text>

    <Button href="[[.Link]]" align="center" class="mt-6"> Verify email </Button>

    <Text class="mt-6 text-sm text-gray-500">
      This link expires on [[.ExpiresAt]] and can only be used once. If you did not create a KinStack account, you can
      safely ignore this email.
    </Text>
  </EmailLayout>
</template>

<template>
  <EmailLayout preheader="You have been invited to join KinStack.">
    <Heading :level="2" class="text-xl font-semibold text-gray-950"> Invitation to join KinStack </Heading>

    <Text>
      A KinStack account has been created for you. Click the button below to activate your account and sign in.
    </Text>

    <Button href="[[.Link]]" align="center" class="mt-6"> Activate account </Button>

    <Text class="mt-6 text-sm text-gray-500">
      This invitation expires on [[.ExpiresAt]] and can only be used once. If you did not expect this invitation, you
      can safely ignore this email.
    </Text>
  </EmailLayout>
</template>

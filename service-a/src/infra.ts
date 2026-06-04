import { applyDecorators } from '@nestjs/common';
import { ApiBearerAuth, ApiExtension } from '@nestjs/swagger';

/**
 * Marks an endpoint as requiring the listed permissions (Infra API scopes).
 * Omit scopes to require only a valid token with no specific permission.
 */
export function InfraAuth(...scopes: string[]) {
  const decorators = [
    ApiBearerAuth('bearer'),
    ...(scopes.length ? [ApiExtension('x-infra-scopes', scopes)] : []),
  ];
  return applyDecorators(...decorators);
}

/**
 * Marks an endpoint as publicly accessible (no token required).
 * Only needed when the service default is protected (infra.auth.protected: "true").
 */
export function InfraPublic() {
  return ApiExtension('x-infra-protected', false);
}

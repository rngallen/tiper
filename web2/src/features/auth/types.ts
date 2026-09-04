export interface TokenResponse {
  email: string;
  name: string;
  issuedAt: string;
  token?: string;
  tokenExpiresAt: string;
  refreshToken?: string;
  refreshTokenExpiresAt: string;
  mustChangePassword: boolean;
  sessionIdleMinutes?: number;
  sessionIdleWarnSeconds?: number;
}

export interface OtpPending {
  message: string;
  token: string;
  issuedAt: string;
  expiredAt: string;
  resendAvailableAt: string;
}

export interface Role {
  id: number;
  name: string;
  description?: string;
  category?: string;
}

export interface User {
  id: string;
  email: string;
  firstName?: string;
  lastName?: string;
  isActive?: boolean;
  isLocked?: boolean;
  isSuperUser?: boolean;
  mustChangePassword?: boolean;
  twoFactorEnabled?: boolean;
  lastLogin?: string;
  canDelete?: boolean;
  roles?: Role[];
  profile?: { phoneNumber?: string; title?: string };
}

export interface MeResponse {
  user: User;
  permissions: string[];
  isSuperUser?: boolean;
  sessionIdleMinutes?: number;
  sessionIdleWarnSeconds?: number;
}

export function isOtpPending(d: TokenResponse | OtpPending): d is OtpPending {
  return (
    typeof (d as OtpPending).token === "string" &&
    typeof (d as OtpPending).message === "string" &&
    typeof (d as TokenResponse).tokenExpiresAt !== "string"
  );
}

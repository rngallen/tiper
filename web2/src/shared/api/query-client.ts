import { QueryClient, type DefaultOptions } from "@tanstack/react-query";
import { isApiError } from "./errors";

const defaultOptions: DefaultOptions = {
  queries: {
    staleTime: 30_000,
    gcTime: 5 * 60_000,
    refetchOnWindowFocus: false,
    retry: (failureCount, error) => {
      // Never retry auth/permission/validation failures; retry transient ones once.
      if (isApiError(error) && error.status < 500 && error.status !== 429) return false;
      return failureCount < 1;
    },
  },
  mutations: {
    retry: false,
  },
};

export function createQueryClient(): QueryClient {
  return new QueryClient({ defaultOptions });
}

export const queryClient = createQueryClient();

import { MutationCache, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import axios from "axios";
import AppRouter from "@/routes/AppRouter";
import { Toaster } from "@/components/ui/toast";
import { useToastStore } from "@/stores/toast-store";

function extractErrorMessage(error: unknown): string {
  if (axios.isAxiosError(error) && typeof error.response?.data?.error === "string") {
    return error.response.data.error;
  }
  return "Something went wrong. Please try again.";
}

const queryClient = new QueryClient({
  mutationCache: new MutationCache({
    onError: (error) => {
      useToastStore.getState().show(extractErrorMessage(error), "error");
    },
  }),
});

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppRouter />
      <Toaster />
    </QueryClientProvider>
  );
}

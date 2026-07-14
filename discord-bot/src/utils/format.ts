/**
 * Normalises a backend payment-method code to a human-friendly label.
 * Handles the codes used across the app: msisdn/promptpay -> PromptPay,
 * bank/bank_account/bankac -> Bank.
 */
export function formatPaymentMethod(method?: string | null): string {
  const lower = (method || "").toLowerCase();
  if (lower === "msisdn" || lower === "promptpay") return "PromptPay";
  if (lower === "bank" || lower === "bank_account" || lower === "bankac") return "Bank";
  return method || "N/A";
}

/** Returns the last 4 digits of an account number, or "????" if unavailable. */
export function maskAccount(account?: string | null): string {
  const digits = (account || "").replace(/\D/g, "");
  return digits.slice(-4) || "????";
}

/** Formats a date string for display, tolerating null/invalid input. */
export function formatDate(dateStr?: string | null): string {
  if (!dateStr) return "N/A";
  const date = new Date(dateStr);
  if (isNaN(date.getTime())) return dateStr;
  return date.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

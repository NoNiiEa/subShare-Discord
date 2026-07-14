/**
 * Centralised, user-safe error handling.
 *
 * Backend errors arrive as an Error whose `.message` is the raw
 * `API Error [500]: ...` string (see api/base.ts) with the parsed payload on
 * `.response`. We must never forward that raw text to Discord users, so all
 * user-facing messages go through here.
 */

export interface ErrorRule {
  /** Case-insensitive substrings to match against the backend error detail. */
  match?: string[];
  /** HTTP status to match against. */
  status?: number;
  /** Friendly message shown to the user when this rule matches. */
  message: string;
}

const GENERIC_MESSAGE = "Something went wrong. Please try again later.";

/**
 * Extracts the backend-provided error detail (never the raw `.message`, which
 * leaks internal `API Error [...]` text).
 */
export function getErrorDetail(err: any): string | undefined {
  return err?.response?.data?.error || err?.response?.data?.message || undefined;
}

/**
 * Maps an error to a friendly, user-safe message using the given rules,
 * falling back to a generic message for anything unmatched.
 */
export function toUserMessage(
  err: any,
  rules: ErrorRule[] = [],
  fallback: string = GENERIC_MESSAGE
): string {
  const detail = (getErrorDetail(err) || "").toLowerCase();
  const status = err?.response?.status;

  for (const rule of rules) {
    const matchesText = rule.match?.some((m) => detail.includes(m.toLowerCase()));
    const matchesStatus = rule.status !== undefined && rule.status === status;
    if (matchesText || matchesStatus) return rule.message;
  }

  return fallback;
}

/** Friendly mappings for /bill paid backend errors. */
export const PAYMENT_ERROR_RULES: ErrorRule[] = [
  { match: ["bill not found"], status: 404, message: "❌ Bill not found. Please select a valid bill." },
  { match: ["bill does not belong to this user"], status: 403, message: "❌ This bill does not belong to you. You cannot pay someone else's bill." },
  { match: ["payment receiver does not match"], message: "❌ Payment receiver does not match group settings. Please check the payment details." },
  { match: ["proof url is required"], message: "❌ Proof URL is required. Please upload a valid slip image." },
  { match: ["member not found in group"], message: "❌ You are not a member of this group." },
  { match: ["bill is already paid"], message: "❌ This bill has already been paid." },
  { match: ["payment slip is too old", "within 30 minutes"], message: "❌ Payment slip is too old. Please upload the slip within 30 minutes of making the transfer." },
  { match: ["no valid qr code"], message: "❌ No valid QR code found in the image. Please ensure the payment slip contains a clear QR code." },
  { match: ["slip qr code has expired", "expired or is invalid"], message: "❌ The slip QR code has expired or is invalid. Please use a recent slip." },
  { match: ["slip has already been used"], status: 409, message: "❌ This slip has already been used. Please use a different payment slip." },
  { match: ["amount in the slip does not match"], message: "❌ The amount in the slip does not match the bill amount. Please verify the payment details." },
  { match: ["valid slip image"], message: "❌ Please upload a valid slip image (JPG/PNG)." },
  { match: ["temporarily unavailable"], message: "⏳ The bank's data is temporarily unavailable. Please try again in a few minutes." },
  { match: ["isn't confirmed by the bank yet"], message: "⏳ This slip isn't confirmed by the bank yet. Please resubmit in a few minutes." },
  { match: ["verification timed out"], status: 504, message: "❌ Verification timed out. Please try again later." },
];

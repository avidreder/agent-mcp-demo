import { privateKeyToAccount, type PrivateKeyAccount } from "viem/accounts";
import { type Hex } from "viem";

export interface WalletInfo {
  address: string;
  account: PrivateKeyAccount;
}

export function createWallet(privateKey: string): WalletInfo {
  // Ensure the private key has 0x prefix
  const formattedKey = privateKey.startsWith("0x")
    ? (privateKey as Hex)
    : (`0x${privateKey}` as Hex);

  const account = privateKeyToAccount(formattedKey);

  return {
    address: account.address,
    account,
  };
}

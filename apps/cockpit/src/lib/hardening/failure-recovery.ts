export function retryWithBackoff(n:number){ return Math.min(30000, 500*(2**n)); }

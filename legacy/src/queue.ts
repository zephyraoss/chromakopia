interface QueuedTask<T> {
  resolve: (value: T) => void;
  reject: (reason?: unknown) => void;
  task: () => Promise<T>;
}

export class RateLimitedQueue {
  private queue: QueuedTask<unknown>[] = [];
  private processing = false;
  private readonly rateLimitMs: number;

  constructor(requestsPerSecond: number) {
    this.rateLimitMs = 1000 / requestsPerSecond;
    this.startProcessing();
  }

  private startProcessing() {
    setInterval(async () => {
      if (this.queue.length === 0) return;

      const item = this.queue.shift();
      if (!item) return;

      try {
        const result = await item.task();
        item.resolve(result);
      } catch (error) {
        item.reject(error);
      }
    }, this.rateLimitMs);
  }

  async enqueue<T>(task: () => Promise<T>): Promise<T> {
    return new Promise((resolve, reject) => {
      this.queue.push({ resolve: resolve as (value: unknown) => void, reject, task });
    });
  }

  get length(): number {
    return this.queue.length;
  }
}

export const queue = new RateLimitedQueue(3);

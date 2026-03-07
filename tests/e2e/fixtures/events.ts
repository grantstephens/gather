export interface TestEventData {
  title: string;
  description: string;
  start_datetime: string;
  end_datetime?: string;
  status: 'draft' | 'pending' | 'published';
  place?: string;
  tags?: string[];
  recurrence_rule?: string;
}

export function createTestEvent(overrides: Partial<TestEventData> = {}): TestEventData {
  const tomorrow = new Date();
  tomorrow.setDate(tomorrow.getDate() + 1);
  tomorrow.setHours(19, 0, 0, 0);

  const endTime = new Date(tomorrow);
  endTime.setHours(22, 0, 0, 0);

  return {
    title: `[TEST] Event ${Date.now()}`,
    description: 'This is a test event created by automated tests.',
    start_datetime: tomorrow.toISOString(),
    end_datetime: endTime.toISOString(),
    status: 'draft',
    ...overrides
  };
}

export const SAMPLE_EVENTS: Record<string, Partial<TestEventData>> = {
  basic: {
    title: '[TEST] Basic Event',
    description: 'A simple test event',
  },
  withMarkdown: {
    title: '[TEST] Markdown Event',
    description: '# Heading\n\n**Bold text** and *italic text*.\n\n- List item 1\n- List item 2',
  },
  recurring: {
    title: '[TEST] Weekly Meeting',
    description: 'Recurring weekly event',
    recurrence_rule: 'FREQ=WEEKLY;COUNT=4',
  }
};

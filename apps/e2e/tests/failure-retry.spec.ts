import { test, expect } from '@playwright/test';

test.describe('Ingestion Failure and Retry Flow', () => {
  const invalidUrl = `http://invalid-url-${Date.now()}.local`;
  let sourceId: string;

  test('should handle ingestion failure and allow retry', async ({ page, request }) => {
    // 1. Create a source with an invalid URL
    console.log(`Creating source with invalid URL: ${invalidUrl}`);
    const createRes = await request.post('http://localhost:8081/sources', {
      data: {
        url: invalidUrl,
        max_depth: 0
      }
    });
    
    if (!createRes.ok()) {
      console.error('Create failed:', await createRes.text());
    }
    expect(createRes.ok()).toBeTruthy();
    const sourceData = await createRes.json();
    sourceId = sourceData.data.id;
    console.log(`Source created with ID: ${sourceId}`);

    // 2. Wait for failure (polling the failed jobs API)
    console.log('Waiting for job to fail...');
    let failedJob = null;
    for (let i = 0; i < 30; i++) { // Poll for 60 seconds max
      const jobsRes = await request.get('http://localhost:8081/jobs/failed');
      expect(jobsRes.ok()).toBeTruthy();
      const jobs = await jobsRes.json();
      failedJob = jobs.find((j: any) => j.source_id === sourceId);
      
      if (failedJob) break;
      await page.waitForTimeout(2000);
    }
    expect(failedJob).toBeTruthy();
    console.log('Job failed as expected:', failedJob);

    // 3. Verify it appears in the UI (Jobs View)
    await page.goto('/jobs');
    // Use filter to target the specific card containing the Source ID
    const jobCard = page.locator('.job-card').filter({ hasText: sourceId });
    await expect(jobCard).toBeVisible();
    await expect(jobCard.locator('.error-msg')).toBeVisible();

    // 4. Retry the job
    console.log('Retrying job...');
    const retryBtn = jobCard.locator('.retry-btn');
    await retryBtn.click();
    
    // 5. Verify it disappears from the list immediately (optimistic UI or refresh)
    // The store removes it from the list on success
    await expect(jobCard).not.toBeVisible();
    console.log('Job removed from list after retry.');

    // 6. Wait for it to fail again (since the URL is still invalid)
    // This confirms the retry actually re-queued the task
    console.log('Waiting for job to fail again...');
    let failedJobRetry = null;
    for (let i = 0; i < 30; i++) {
      const jobsRes = await request.get('http://localhost:8081/jobs/failed');
      expect(jobsRes.ok()).toBeTruthy();
      const jobs = await jobsRes.json();
      failedJobRetry = jobs.find((j: any) => j.source_id === sourceId);
      
      if (failedJobRetry) {
        // Ensure it's a new failure event (optional: check timestamp or just existence)
        // Since we deleted the old one, existence implies a new one
        break;
      }
      await page.waitForTimeout(2000);
    }
    expect(failedJobRetry).toBeTruthy();
    console.log('Job failed again after retry.');

    // Cleanup
    await request.delete(`http://localhost:8081/sources/${sourceId}`);
  });
});

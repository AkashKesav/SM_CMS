import { useCallback, useEffect, useState } from 'react';
import { adminApi } from '@/lib/api';
import { StudentChangeRequest } from '@/types/schema';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { formatColumnName } from '@/utils/fieldMapper';
import { formatDate } from '@/lib/utils';
import { Check, ClipboardCheck, Loader2, X } from 'lucide-react';
import toast from 'react-hot-toast';

export function AdminReviewPage() {
  const [status, setStatus] = useState('pending');
  const [requests, setRequests] = useState<StudentChangeRequest[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [activeId, setActiveId] = useState<string | null>(null);

  const loadRequests = useCallback(async () => {
    setIsLoading(true);
    const response = await adminApi.listChangeRequests(status);
    if (response.success && response.data) {
      setRequests(response.data);
    } else {
      toast.error(response.error || 'Failed to load review queue');
    }
    setIsLoading(false);
  }, [status]);

  useEffect(() => {
    loadRequests();
  }, [loadRequests]);

  const review = async (id: string, action: 'approve' | 'reject') => {
    const note = action === 'reject' ? window.prompt('Reason for rejection?') || '' : '';
    setActiveId(id);
    const response = action === 'approve'
      ? await adminApi.approveChangeRequest(id)
      : await adminApi.rejectChangeRequest(id, note);
    if (response.success) {
      toast.success(action === 'approve' ? 'Request approved' : 'Request rejected');
      await loadRequests();
    } else {
      toast.error(response.error || response.details || `Failed to ${action} request`);
    }
    setActiveId(null);
  };

  return (
    <div className="cms-page">
      <div className="cms-page-header md:flex-row md:items-center">
        <div>
          <div className="cms-eyebrow mb-2">
            <ClipboardCheck className="h-3.5 w-3.5" />
            Review queue
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">Student Change Requests</h1>
          <p className="mt-1 text-sm text-muted-foreground">{requests.length} requests in this view</p>
        </div>
        <div className="w-full md:w-48">
          <Select value={status} onValueChange={setStatus}>
            <SelectTrigger>
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="approved">Approved</SelectItem>
              <SelectItem value="rejected">Rejected</SelectItem>
              <SelectItem value="all">All</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-primary" />
        </div>
      ) : (
        <div className="space-y-4">
          {requests.map((request) => (
            <Card key={request.id}>
              <CardHeader>
                <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                  <div>
                    <CardTitle className="text-lg">
                      {formatColumnName(request.target_table)} | {formatColumnName(request.request_type)}
                    </CardTitle>
                    <CardDescription>
                      Submitted {formatDate(request.created_at)} by{' '}
                      {request.requester_name || request.requester_roll_no
                        ? <span className="font-medium text-foreground">
                            {request.requester_name || 'Unknown'}
                            {request.requester_roll_no && (
                              <span className="ml-1 text-muted-foreground">({request.requester_roll_no})</span>
                            )}
                          </span>
                        : <span className="font-mono text-xs">{request.requester_user_id.slice(0, 8)}…</span>
                      }
                    </CardDescription>
                    {requestSummary(request) && (
                      <p className="mt-2 text-sm text-muted-foreground">{requestSummary(request)}</p>
                    )}
                  </div>
                  <StatusBadge status={request.status} />
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 lg:grid-cols-2">
                  <DiffBlock title="Current" data={request.current_data || {}} />
                  <DiffBlock title="Proposed" data={request.proposed_data || {}} />
                </div>
                {request.admin_note && (
                  <div className="rounded-md border bg-muted p-3 text-sm">{request.admin_note}</div>
                )}
                {request.status === 'pending' && (
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Button onClick={() => review(request.id, 'approve')} disabled={activeId === request.id}>
                      <Check className="mr-2 h-4 w-4" />
                      Approve
                    </Button>
                    <Button variant="outline" onClick={() => review(request.id, 'reject')} disabled={activeId === request.id}>
                      <X className="mr-2 h-4 w-4" />
                      Reject
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
          {requests.length === 0 && (
            <Card>
              <CardContent className="py-12 text-center text-sm text-muted-foreground">
                No requests found.
              </CardContent>
            </Card>
          )}
        </div>
      )}
    </div>
  );
}

function DiffBlock({ title, data }: { title: string; data: Record<string, any> }) {
  return (
    <div className="rounded-md border">
      <div className="border-b px-3 py-2 text-sm font-medium">{title}</div>
      <pre className="max-h-80 overflow-auto p-3 text-xs">
        {JSON.stringify(data, null, 2)}
      </pre>
    </div>
  );
}

function StatusBadge({ status }: { status: StudentChangeRequest['status'] }) {
  const classes = {
    pending: 'border-amber-200 bg-amber-50 text-amber-700',
    approved: 'border-emerald-200 bg-emerald-50 text-emerald-700',
    rejected: 'border-red-200 bg-red-50 text-red-700',
  };
  return <Badge variant="outline" className={classes[status]}>{formatColumnName(status)}</Badge>;
}

function requestSummary(request: StudentChangeRequest) {
  const data = request.proposed_data || {};
  if (request.target_table === 'achievement_members') {
    const title = data.achievement_title || data.achievement_id || 'achievement';
    const student = data.student_name || data.student_roll_no || data.student_id || 'student';
    return `Add contributor ${student} to ${title}`;
  }
  if (request.target_table === 'project_members') {
    const title = data.project_title || data.project_id || 'project';
    const student = data.student_name || data.student_roll_no || data.student_id || 'student';
    return `Add contributor ${student} to ${title}`;
  }
  if (request.target_table === 'achievements' && data.title) {
    return `Achievement: ${data.title}`;
  }
  if (request.target_table === 'students') {
    return data.name || data.roll_no || '';
  }
  if (data.title) {
    return String(data.title);
  }
  if (data.name) {
    return String(data.name);
  }
  return '';
}

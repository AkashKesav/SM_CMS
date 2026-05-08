import { useCallback, useEffect, useMemo, useState } from 'react';
import { storageApi, studentApi } from '@/lib/api';
import { StudentChangeRequest, StudentContributor, StudentSearchResult, StudentWorkspace } from '@/types/schema';
import { Button } from '@/components/ui/Button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { Input } from '@/components/ui/Input';
import { Label } from '@/components/ui/Label';
import { Textarea } from '@/components/ui/Textarea';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/Select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/Tabs';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/Dialog';
import { formatColumnName } from '@/utils/fieldMapper';
import { formatDate } from '@/lib/utils';
import { CheckCircle2, Clock3, FileText, Loader2, Pencil, Plus, Send, UserRound, X } from 'lucide-react';
import toast from 'react-hot-toast';

const linkedTables = [
  'achievements',
  'projects',
  'achievement_members',
  'student_tags',
  'project_members',
  'hof_competition_members',
  'hof_coordinators',
  'hof_internships',
];

const editableStudentFields = ['name', 'roll_no', 'email', 'gender', 'batch_id', 'linkedin', 'image_url', 'bio'];

export function StudentWorkspacePage() {
  const [workspace, setWorkspace] = useState<StudentWorkspace | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [profileData, setProfileData] = useState<Record<string, any>>({});
  const [linkedTable, setLinkedTable] = useState('achievements');
  const [linkedForm, setLinkedForm] = useState<Record<string, any>>({});
  const [linkedContributors, setLinkedContributors] = useState<StudentContributor[]>([]);
  const [contributorsLoading, setContributorsLoading] = useState(false);
  const [editing, setEditing] = useState<{ table: string; record: Record<string, any>; json: string } | null>(null);

  const loadWorkspace = useCallback(async () => {
    setIsLoading(true);
    const response = await studentApi.me();
    if (response.success && response.data) {
      setWorkspace(response.data);
      const student = response.data.student || {};
      setProfileData(
        editableStudentFields.reduce<Record<string, any>>((acc, field) => {
          acc[field] = student[field] ?? '';
          return acc;
        }, {})
      );
    } else {
      toast.error(response.error || 'Failed to load student workspace');
    }
    setIsLoading(false);
  }, []);

  useEffect(() => {
    loadWorkspace();
  }, [loadWorkspace]);

  useEffect(() => {
    const recordType = linkedTable === 'achievement_members'
      ? 'achievement'
      : linkedTable === 'project_members'
        ? 'project'
        : null;
    const recordId = linkedTable === 'achievement_members'
      ? linkedForm.achievement_id
      : linkedTable === 'project_members'
        ? linkedForm.project_id
        : '';

    if (!recordType || !recordId) {
      setLinkedContributors([]);
      return;
    }

    let active = true;
    setContributorsLoading(true);
    studentApi.getContributors(recordType, String(recordId)).then((response) => {
      if (!active) return;
      if (response.success && response.data) {
        setLinkedContributors(response.data);
      } else {
        setLinkedContributors([]);
      }
      setContributorsLoading(false);
    });

    return () => {
      active = false;
    };
  }, [linkedTable, linkedForm.achievement_id, linkedForm.project_id]);

  const pendingCount = useMemo(
    () => workspace?.requests.filter((request) => request.status === 'pending').length || 0,
    [workspace]
  );

  const submitProfileRequest = async (event: React.FormEvent) => {
    event.preventDefault();
    setIsSubmitting(true);
    const proposed = cleanData(profileData);
    const response = await studentApi.createRequest({
      request_type: workspace?.student ? 'update' : 'create',
      target_table: 'students',
      proposed_data: proposed,
    });
    if (response.success) {
      toast.success('Profile changes sent for review');
      await loadWorkspace();
    } else {
      toast.error(response.error || response.details || 'Failed to submit changes');
    }
    setIsSubmitting(false);
  };

  const submitLinkedRequest = async (event: React.FormEvent) => {
    event.preventDefault();
    setIsSubmitting(true);
    const cleaned = cleanData(linkedForm);
    let response;

    if (linkedTable === 'achievement_members') {
      if (!cleaned.achievement_id) {
        toast.error('Achievement is required');
        setIsSubmitting(false);
        return;
      }
      response = await studentApi.requestAchievementContributor({
        achievement_id: String(cleaned.achievement_id),
        student_id: cleaned.student_id ? String(cleaned.student_id) : undefined,
      });
    } else if (linkedTable === 'project_members') {
      if (!cleaned.project_id) {
        toast.error('Project is required');
        setIsSubmitting(false);
        return;
      }
      response = await studentApi.requestProjectContributor({
        project_id: String(cleaned.project_id),
        student_id: cleaned.student_id ? String(cleaned.student_id) : undefined,
        role: cleaned.role ? String(cleaned.role) : undefined,
      });
    } else {
      const requestType = linkLikeTable(linkedTable) ? 'link' : 'create';
      response = await studentApi.createRequest({
        request_type: requestType,
        target_table: linkedTable,
        proposed_data: cleaned,
      });
    }

    if (response?.success) {
      toast.success('Request submitted for admin review');
      setLinkedForm({});
      await loadWorkspace();
    } else {
      toast.error(response?.details || response?.error || 'Failed to submit request');
    }
    setIsSubmitting(false);
  };

  const submitExistingChange = async (table: string, record: Record<string, any>, requestType: StudentChangeRequest['request_type'], proposedData?: Record<string, any>) => {
    setIsSubmitting(true);
    const response = await studentApi.createRequest({
      request_type: requestType,
      target_table: table,
      target_key: getTargetKey(table, record),
      proposed_data: proposedData || record,
    });
    if (response.success) {
      toast.success('Request sent for review');
      setEditing(null);
      await loadWorkspace();
    } else {
      toast.error(response.error || response.details || 'Failed to submit request');
    }
    setIsSubmitting(false);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  return (
    <div className="cms-page">
      <div className="cms-page-header md:flex-row md:items-center">
        <div className="min-w-0">
          <div className="cms-eyebrow mb-2">
            <UserRound className="h-3.5 w-3.5" />
            Student workspace
          </div>
          <h1 className="text-2xl font-semibold tracking-tight">My Info</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {workspace?.student ? `${workspace.student.name || 'Student'} | ${workspace.student.roll_no || 'No roll number'}` : 'Your account is waiting for an approved student link.'}
          </p>
        </div>
        <div className="flex gap-2">
          <Badge variant={pendingCount > 0 ? 'default' : 'outline'} className="gap-1.5">
            <Clock3 className="h-3.5 w-3.5" />
            {pendingCount} pending
          </Badge>
          {workspace?.profile.student_id && (
            <Badge variant="outline" className="gap-1.5 border-emerald-200 bg-emerald-50 text-emerald-700">
              <CheckCircle2 className="h-3.5 w-3.5" />
              Linked
            </Badge>
          )}
        </div>
      </div>

      <Tabs defaultValue="profile" className="space-y-4">
        <TabsList className="flex h-auto w-full flex-wrap justify-start">
          <TabsTrigger value="profile">My Info</TabsTrigger>
          <TabsTrigger value="linked">Linked Records</TabsTrigger>
          <TabsTrigger value="requests">Requests</TabsTrigger>
        </TabsList>

        <TabsContent value="profile">
          <Card>
            <CardHeader>
              <CardTitle>Student Record</CardTitle>
              <CardDescription>Changes are sent to admins before they go live.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={submitProfileRequest} className="cms-field-grid">
                <TextField label="Name" value={profileData.name || ''} onChange={(value) => setProfileData({ ...profileData, name: value })} required />
                <TextField label="Roll number" value={profileData.roll_no || ''} onChange={(value) => setProfileData({ ...profileData, roll_no: value })} required />
                <TextField label="Email" type="email" value={profileData.email || ''} onChange={(value) => setProfileData({ ...profileData, email: value })} />
                <TextField label="LinkedIn" value={profileData.linkedin || ''} onChange={(value) => setProfileData({ ...profileData, linkedin: value })} />
                <ImageUploaderField
                  label="Image"
                  value={profileData.image_url || ''}
                  onChange={(value) => setProfileData({ ...profileData, image_url: value })}
                  bucket="high_quality_image"
                />
                <div className="space-y-2">
                  <Label>Gender</Label>
                  <Select value={profileData.gender || '__none__'} onValueChange={(value) => setProfileData({ ...profileData, gender: value === '__none__' ? '' : value })}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select gender" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__none__">Not set</SelectItem>
                      <SelectItem value="male">Male</SelectItem>
                      <SelectItem value="female">Female</SelectItem>
                      <SelectItem value="other">Other</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <ReferenceSelect
                  label="Batch"
                  value={profileData.batch_id || ''}
                  options={workspace?.references.batches || []}
                  labelField="year"
                  onChange={(value) => setProfileData({ ...profileData, batch_id: value })}
                />
                <div className="space-y-2 md:col-span-2">
                  <Label>Bio</Label>
                  <Textarea
                    value={profileData.bio || ''}
                    onChange={(event) => setProfileData({ ...profileData, bio: event.target.value })}
                    rows={5}
                  />
                </div>
                <div className="md:col-span-2">
                  <Button type="submit" disabled={isSubmitting}>
                    <Send className="mr-2 h-4 w-4" />
                    Submit For Review
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="linked" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Plus className="h-5 w-5" />
                Add Linked Info
              </CardTitle>
              <CardDescription>Choose existing catalog items when linking projects, tags, batches, or competitions.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={submitLinkedRequest} className="cms-field-grid">
                <div className="space-y-2">
                  <Label>Type</Label>
                  <Select value={linkedTable} onValueChange={(value) => { setLinkedTable(value); setLinkedForm({}); }}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {linkedTables.map((table) => {
                        let displayName = formatColumnName(table);
                        if (table === 'achievements') displayName = 'Create New Achievement';
                        if (table === 'projects') displayName = 'Create New Project';
                        if (table === 'achievement_members') displayName = 'Link Existing Achievement';
                        if (table === 'project_members') displayName = 'Link Existing Project';
                        return (
                          <SelectItem key={table} value={table}>{displayName}</SelectItem>
                        );
                      })}
                    </SelectContent>
                  </Select>
                </div>
                <LinkedRequestFields
                  table={linkedTable}
                  data={linkedForm}
                  references={workspace?.references || {}}
                  contributors={linkedContributors}
                  contributorsLoading={contributorsLoading}
                  onChange={setLinkedForm}
                />
                <div className="md:col-span-2">
                  <Button type="submit" disabled={isSubmitting || !workspace?.profile.student_id}>
                    <Send className="mr-2 h-4 w-4" />
                    Submit Linked Request
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>

          <div className="grid gap-4">
            {linkedTables
              .filter((table) => !['projects', 'achievements'].includes(table))
              .map((table) => {
                let cardTitle = formatColumnName(table);
                if (table === 'achievement_members') cardTitle = 'Achievements';
                if (table === 'project_members') cardTitle = 'Projects';
                
                return (
                  <Card key={table}>
                    <CardHeader>
                      <CardTitle className="text-base">{cardTitle}</CardTitle>
                      <CardDescription>{workspace?.linked_records[table]?.length || 0} records</CardDescription>
                    </CardHeader>
                <CardContent className="space-y-3">
                  {(workspace?.linked_records[table] || []).map((record, index) => (
                    <div key={`${table}-${index}`} className="flex flex-col gap-3 rounded-md border bg-muted/20 p-3 md:flex-row md:items-center md:justify-between">
                      <div className="min-w-0">
                        <p className="truncate text-sm font-medium">{recordTitle(table, record, workspace?.references || {})}</p>
                        <p className="truncate text-xs text-muted-foreground">{recordSubtitle(record)}</p>
                      </div>
                      <div className="flex gap-2">
                        <Button variant="outline" size="sm" onClick={() => {
                          let editTable = table;
                          let editRecord = record;
                          
                          if (table === 'project_members' && record.project_id) {
                            const fullProject = workspace?.references.projects?.find((p: any) => p.id === record.project_id);
                            if (fullProject) {
                              editTable = 'projects';
                              editRecord = fullProject;
                            }
                          } else if (table === 'achievement_members' && record.achievement_id) {
                            const fullAchievement = workspace?.references.achievements?.find((a: any) => a.id === record.achievement_id);
                            if (fullAchievement) {
                              editTable = 'achievements';
                              editRecord = fullAchievement;
                            }
                          }
                          
                          setEditing({ table: editTable, record: editRecord, data: { ...editRecord } } as any);
                        }}>
                          <Pencil className="mr-2 h-3.5 w-3.5" />
                          Edit
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => submitExistingChange(table, record, linkLikeTable(table) ? 'unlink' : 'delete', {})}
                          disabled={isSubmitting}
                        >
                          Remove
                        </Button>
                      </div>
                    </div>
                  ))}
                  {(workspace?.linked_records[table] || []).length === 0 && (
                    <p className="text-sm text-muted-foreground">No records yet.</p>
                  )}
                </CardContent>
              </Card>
            )})}
          </div>
        </TabsContent>

        <TabsContent value="requests">
          <RequestsList requests={workspace?.requests || []} />
        </TabsContent>
      </Tabs>

      <Dialog open={!!editing} onOpenChange={(open) => !open && setEditing(null)}>
        <DialogContent className="max-h-[90vh] max-w-3xl overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Edit {editing?.table ? formatColumnName(editing.table) : 'Record'}</DialogTitle>
            <DialogDescription>Update the details below. Your changes will be reviewed by an admin.</DialogDescription>
          </DialogHeader>
          {editing && (
            <form
              className="space-y-6"
              onSubmit={(event) => {
                event.preventDefault();
                submitExistingChange(editing.table, editing.record, 'update', (editing as any).data);
              }}
            >
              <div className="cms-field-grid">
                <LinkedRequestFields
                  table={editing.table}
                  data={(editing as any).data}
                  references={workspace?.references || {}}
                  contributors={linkedContributors}
                  contributorsLoading={contributorsLoading}
                  onChange={(newData) => setEditing({ ...editing, data: newData } as any)}
                />
              </div>
              <div className="flex justify-end gap-3">
                <Button type="button" variant="outline" onClick={() => setEditing(null)}>
                  Cancel
                </Button>
                <Button type="submit" disabled={isSubmitting}>
                  <Send className="mr-2 h-4 w-4" />
                  Submit Edit
                </Button>
              </div>
            </form>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function TextField({ label, value, onChange, type = 'text', required = false }: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  required?: boolean;
}) {
  const id = label.toLowerCase().replace(/\s+/g, '-');
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input id={id} type={type} value={value} onChange={(event) => onChange(event.target.value)} required={required} />
    </div>
  );
}

function ImageUploaderField({ label, value, onChange, placeholder, bucket }: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  bucket?: string;
}) {
  const [isUploading, setIsUploading] = useState(false);

  const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    setIsUploading(true);
    try {
      const targetBucket = bucket || import.meta.env.VITE_SUPABASE_STORAGE_BUCKET || 'cms-media';
      const signedUrlResp = await storageApi.getSignedUrl(targetBucket, file.name, file.type);

      if (signedUrlResp.success && signedUrlResp.data?.signed_url) {
        const publicUrl = await storageApi.uploadToSupabase(signedUrlResp.data.signed_url, file);
        onChange(publicUrl);
      } else {
        const url = URL.createObjectURL(file);
        onChange(url);
      }
    } catch (error) {
      console.error('Upload failed:', error);
      const url = URL.createObjectURL(file);
      onChange(url);
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      {value ? (
        <div className="relative">
          <img
            src={value}
            alt="Preview"
            className="h-32 w-full rounded-md object-cover"
            onError={(e) => {
              (e.target as HTMLImageElement).style.display = 'none';
            }}
          />
          <button
            type="button"
            className="absolute right-1 top-1 rounded-full bg-destructive p-1 text-destructive-foreground"
            onClick={() => onChange('')}
          >
            <X className="h-3 w-3" />
          </button>
        </div>
      ) : (
        <label className="flex h-24 w-full cursor-pointer flex-col items-center justify-center rounded-md border-2 border-dashed hover:bg-accent">
          <div className="flex flex-col items-center justify-center py-2">
            {isUploading ? (
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            ) : (
              <Plus className="h-6 w-6 text-muted-foreground" />
            )}
            <p className="mt-1 text-xs text-muted-foreground">
              {isUploading ? 'Uploading...' : 'Click to upload image'}
            </p>
          </div>
          <input
            type="file"
            accept="image/*"
            className="hidden"
            onChange={handleFileSelect}
            disabled={isUploading}
          />
        </label>
      )}
      <Input
        type="url"
        value={value || ''}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder || 'Or paste image URL'}
        className="text-sm"
      />
    </div>
  );
}

function ReferenceSelect({ label, value, options, labelField, onChange }: {
  label: string;
  value: string;
  options: Record<string, any>[];
  labelField: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <Select value={value || '__none__'} onValueChange={(next) => onChange(next === '__none__' ? '' : next)}>
        <SelectTrigger>
          <SelectValue placeholder={`Select ${label.toLowerCase()}`} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="__none__">Not set</SelectItem>
          {options.map((option) => (
            <SelectItem key={String(option.id)} value={String(option.id)}>
              {String(option[labelField] ?? option.name ?? option.title ?? option.id)}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function LinkedRequestFields({ table, data, references, contributors, contributorsLoading, onChange, isEditing = false }: {
  table: string;
  data: Record<string, any>;
  references: Record<string, Record<string, any>[]>;
  contributors: StudentContributor[];
  contributorsLoading: boolean;
  onChange: (data: Record<string, any>) => void;
  isEditing?: boolean;
}) {
  switch (table) {
    case 'achievement_members':
      return (
        <>
          {!isEditing && (
            <ReferenceSelect label="Achievement" value={data.achievement_id || ''} options={references.achievements || []} labelField="title" onChange={(value) => onChange({ ...data, achievement_id: value })} />
          )}
          {!isEditing && (
            <StudentSearchField
              value={data.student_id || ''}
              onChange={(value) => onChange({ ...data, student_id: value })}
              excludedIds={contributors.map((item) => item.id)}
              existing={contributors}
              isLoading={contributorsLoading}
            />
          )}
        </>
      );
    case 'student_tags':
      return <ReferenceSelect label="Tag" value={data.tag_id || ''} options={references.tags || []} labelField="name" onChange={(value) => onChange({ ...data, tag_id: value })} />;
    case 'projects':
      return (
        <>
          <TextField label="Title" value={data.title || ''} onChange={(value) => onChange({ ...data, title: value })} required />
          <ReferenceSelect label="Category" value={data.category_id || ''} options={references.project_categories || []} labelField="name" onChange={(value) => onChange({ ...data, category_id: value })} />
          <TextField label="Project date" type="date" value={data.project_date || ''} onChange={(value) => onChange({ ...data, project_date: value })} />
          <ImageUploaderField label="Project image" value={data.image_url || ''} onChange={(value) => onChange({ ...data, image_url: value })} bucket="student_project" />
          <div className="space-y-2 md:col-span-2">
            <Label>Description</Label>
            <Textarea value={data.description || ''} onChange={(event) => onChange({ ...data, description: event.target.value })} required rows={4} />
          </div>
        </>
      );
    case 'project_members':
      return (
        <>
          {!isEditing && (
            <ReferenceSelect label="Project" value={data.project_id || ''} options={references.projects || []} labelField="title" onChange={(value) => onChange({ ...data, project_id: value })} />
          )}
          <TextField label="Role" value={data.role || ''} onChange={(value) => onChange({ ...data, role: value })} />
          {!isEditing && (
            <StudentSearchField
              value={data.student_id || ''}
              onChange={(value) => onChange({ ...data, student_id: value })}
              excludedIds={contributors.map((item) => item.id)}
              existing={contributors}
              isLoading={contributorsLoading}
            />
          )}
        </>
      );
    case 'hof_competition_members':
      return (
        <>
          <ReferenceSelect label="Competition" value={data.competition_id || ''} options={references.hof_competitions || []} labelField="name" onChange={(value) => onChange({ ...data, competition_id: value })} />
          <TextField label="Team role" value={data.team_role || ''} onChange={(value) => onChange({ ...data, team_role: value })} />
        </>
      );
    case 'hof_coordinators':
      return (
        <>
          <TextField label="Role" value={data.role || ''} onChange={(value) => onChange({ ...data, role: value })} required />
          <TextField label="Society" value={data.society || ''} onChange={(value) => onChange({ ...data, society: value })} required />
          <TextField label="Display order" type="number" value={data.display_order || ''} onChange={(value) => onChange({ ...data, display_order: value })} />
        </>
      );
    case 'hof_internships':
      return (
        <>
          <TextField label="Company" value={data.company || ''} onChange={(value) => onChange({ ...data, company: value })} required />
          <TextField label="Role" value={data.role || ''} onChange={(value) => onChange({ ...data, role: value })} required />
          <TextField label="Display order" type="number" value={data.display_order || ''} onChange={(value) => onChange({ ...data, display_order: value })} />
        </>
      );
    default:
      return (
        <>
          <TextField label="Title" value={data.title || ''} onChange={(value) => onChange({ ...data, title: value })} required />
          <TextField label="Category" value={data.category || ''} onChange={(value) => onChange({ ...data, category: value })} />
          <TextField label="Achievement date" type="date" value={data.achievement_date || ''} onChange={(value) => onChange({ ...data, achievement_date: value })} />
          <ImageUploaderField label="Achievement image" value={data.image_url || ''} onChange={(value) => onChange({ ...data, image_url: value })} bucket="high_quality_image" />
          <div className="space-y-2 md:col-span-2">
            <Label>Description</Label>
            <Textarea value={data.description || ''} onChange={(event) => onChange({ ...data, description: event.target.value })} />
          </div>
        </>
      );
  }
}

function StudentSearchField({
  value,
  onChange,
  excludedIds = [],
  existing = [],
  isLoading = false,
}: {
  value: string;
  onChange: (value: string) => void;
  excludedIds?: string[];
  existing?: StudentContributor[];
  isLoading?: boolean;
}) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<StudentSearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [selected, setSelected] = useState<StudentSearchResult | null>(null);

  useEffect(() => {
    if (!value) {
      setSelected(null);
    }
  }, [value]);

  const clearSelection = () => {
    setSelected(null);
    onChange('');
  };

  const runSearch = async () => {
    const trimmed = query.trim();
    if (!trimmed) {
      setResults([]);
      return;
    }
    setIsSearching(true);
    const response = await studentApi.searchStudents(trimmed, 20);
    if (response.success && response.data) {
      const filtered = response.data.filter((student) => !excludedIds.includes(student.id));
      setResults(filtered);
    } else {
      toast.error(response.error || 'Failed to search students');
    }
    setIsSearching(false);
  };

  const handleSelect = (student: StudentSearchResult) => {
    setSelected(student);
    setResults([]);
    setQuery('');
    onChange(student.id);
  };

  return (
    <div className="space-y-2 md:col-span-2">
      <Label>Contributor (optional)</Label>
      <p className="text-xs text-muted-foreground">Leave empty to request yourself. Pick a student to request adding them.</p>
      {selected ? (
        <div className="flex items-center justify-between rounded-md border bg-muted/20 px-3 py-2">
          <div>
            <p className="text-sm font-medium">{selected.name}</p>
            <p className="text-xs text-muted-foreground">{selected.roll_no}</p>
          </div>
          <Button type="button" variant="outline" size="sm" onClick={clearSelection}>Clear</Button>
        </div>
      ) : (
        <div className="space-y-2">
          <div className="flex flex-col gap-2 md:flex-row">
            <Input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Search by name or roll no"
            />
            <Button type="button" variant="outline" onClick={runSearch} disabled={isSearching}>
              {isSearching ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
              Search
            </Button>
          </div>
          {results.length > 0 && (
            <div className="max-h-52 overflow-auto rounded-md border bg-background">
              {results.map((student) => (
                <button
                  key={student.id}
                  type="button"
                  onClick={() => handleSelect(student)}
                  className="flex w-full items-center justify-between px-3 py-2 text-left text-sm hover:bg-muted/50"
                >
                  <span>{student.name}</span>
                  <span className="text-xs text-muted-foreground">{student.roll_no}</span>
                </button>
              ))}
            </div>
          )}
          {!isSearching && query && results.length === 0 && (
            <p className="text-xs text-muted-foreground">No matching students available to add.</p>
          )}
        </div>
      )}
      <div className="space-y-2">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>Existing contributors</span>
          {isLoading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
        </div>
        {existing.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {existing.map((student) => (
              <span key={student.id} className="rounded-full border bg-muted/30 px-2 py-0.5 text-xs">
                {student.name} ({student.roll_no})
              </span>
            ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No contributors found yet.</p>
        )}
      </div>
    </div>
  );
}

function RequestsList({ requests }: { requests: StudentChangeRequest[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <FileText className="h-5 w-5" />
          Requests
        </CardTitle>
        <CardDescription>{requests.length} submitted changes</CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {requests.map((request) => (
          <div key={request.id} className="rounded-md border p-3">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <p className="font-medium">{formatColumnName(request.target_table)} | {formatColumnName(request.request_type)}</p>
                <p className="text-xs text-muted-foreground">{formatDate(request.created_at)}</p>
              </div>
              <StatusBadge status={request.status} />
            </div>
            {requestSummary(request) && (
              <p className="mt-2 text-sm text-muted-foreground">{requestSummary(request)}</p>
            )}
            {request.admin_note && <p className="mt-2 text-sm text-muted-foreground">{request.admin_note}</p>}
          </div>
        ))}
        {requests.length === 0 && <p className="text-sm text-muted-foreground">No requests yet.</p>}
      </CardContent>
    </Card>
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

function cleanData(data: Record<string, any>) {
  return Object.entries(data).reduce<Record<string, any>>((acc, [key, value]) => {
    if (value === '') {
      acc[key] = null;
    } else if (key === 'display_order' && value !== null && value !== undefined) {
      acc[key] = Number(value);
    } else {
      acc[key] = value;
    }
    return acc;
  }, {});
}

function linkLikeTable(table: string) {
  return ['achievement_members', 'student_tags', 'project_members', 'hof_competition_members'].includes(table);
}

function getTargetKey(table: string, record: Record<string, any>) {
  if (table === 'achievements') return { id: record.id };
  if (table === 'achievement_members') return { student_id: record.student_id, achievement_id: record.achievement_id };
  if (table === 'student_tags') return { student_id: record.student_id, tag_id: record.tag_id };
  if (table === 'project_members') return { student_id: record.student_id, project_id: record.project_id };
  if (table === 'hof_competition_members') return { student_id: record.student_id, competition_id: record.competition_id };
  return { id: record.id };
}

function recordTitle(table: string, record: Record<string, any>, references: Record<string, Record<string, any>[]>) {
  if (table === 'achievements') return `Achievement: ${record.title || 'Untitled'}`;
  if (table === 'achievement_members') return record.achievement_title ? `Achievement (Linked): ${record.achievement_title}` : `Contributor for: ${lookupReference(references.achievements, record.achievement_id, 'title')}`;
  if (table === 'student_tags') return `Tag: ${lookupReference(references.tags, record.tag_id, 'name')}`;
  if (table === 'project_members') return record.project_title ? `Project (Linked): ${record.project_title}` : `Project: ${lookupReference(references.projects, record.project_id, 'title')}`;
  if (table === 'hof_competition_members') return `Competition: ${lookupReference(references.hof_competitions, record.competition_id, 'name')}`;
  return String(record.title || record.company || record.society || record.role || record.id || 'Record');
}

function recordSubtitle(record: Record<string, any>) {
  const subtitle = Object.entries(record)
    .filter(([key, value]) => !key.includes('title') && !key.includes('category') && value !== null && value !== undefined && value !== '')
    .slice(0, 4)
    .map(([key, value]) => `${key}: ${String(value)}`)
    .join(' | ');
  
  if (record.achievement_category || record.project_category) {
    return `${record.achievement_category || record.project_category} | ${subtitle}`;
  }
  return subtitle;
}

function lookupReference(options: Record<string, any>[] = [], id: string, labelField: string) {
  const found = options.find((option) => String(option.id) === String(id));
  return String(found?.[labelField] || id || 'Not set');
}

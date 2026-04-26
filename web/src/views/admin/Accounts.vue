<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import * as accountApi from '@/api/accounts'
import { http } from '@/api/http'
import * as proxyApi from '@/api/proxies'
import AccountImportDialog from '@/components/admin/account-import/AccountImportDialog.vue'
import type {
  DialogSubmitPayload,
  FileImportModel,
  ImportDialogResultRow,
  SelectOption,
} from '@/components/admin/account-import/types'
import { formatDateShort } from '@/utils/format'

// ========== 列表 & 筛选 ==========
const loading = ref(false)
const filter = reactive<{ status?: string; keyword?: string }>({ status: '', keyword: '' })
const rows = ref<accountApi.Account[]>([])
const total = ref(0)
const pager = reactive({ page: 1, page_size: 10 })
const proxies = ref<proxyApi.Proxy[]>([])
const proxyOptions = computed<SelectOption[]>(() =>
  proxies.value.map((item) => ({
    value: item.id,
    label: `#${item.id} ${item.remark || item.host}:${item.port}`,
    disabled: !item.enabled,
  })),
)
const poolOptions = ref<SelectOption[]>([])

interface AccountPoolListItem {
  id: number
  code?: string
  name?: string
  enabled?: boolean
}

async function fetchPools() {
  try {
    const data = await http.get<any, { items?: AccountPoolListItem[] }>('/api/admin/account-pools')
    poolOptions.value = (data.items || []).map((item) => ({
      value: item.id,
      label: item.name ? `${item.name} (${item.code || `#${item.id}`})` : item.code || `#${item.id}`,
      disabled: item.enabled === false,
    }))
  } catch {
    /* noop */
  }
}

async function fetchList() {
  loading.value = true
  try {
    const data = await accountApi.listAccounts({
      page: pager.page,
      page_size: pager.page_size,
      status: filter.status || undefined,
      keyword: filter.keyword || undefined,
    })
    rows.value = data.list || []
    total.value = data.total || 0
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function fetchProxies() {
  try {
    const d = await proxyApi.listProxies({ page: 1, page_size: 500 })
    proxies.value = (d.list || []).filter((p) => p.enabled)
  } catch {
    /* noop */
  }
}

function onSearch() {
  pager.page = 1
  fetchList()
}
function onReset() {
  filter.status = ''
  filter.keyword = ''
  pager.page = 1
  fetchList()
}

// ========== 自动刷新开关 ==========
const autoRefreshEnabled = ref(false)
const autoRefreshSaving = ref(false)

async function loadAutoRefresh() {
  try {
    const cfg = await accountApi.getAutoRefresh()
    autoRefreshEnabled.value = !!cfg.enabled
  } catch {
    /* noop */
  }
}
async function onToggleAutoRefresh(val: boolean | string | number) {
  const enabled = !!val
  autoRefreshSaving.value = true
  try {
    await accountApi.setAutoRefresh(enabled)
    autoRefreshEnabled.value = enabled
    ElMessage.success(
      enabled
        ? '已开启自动刷新:AT 距离过期 < 1 天时自动续期,失效/可疑账号不刷新'
        : '已关闭自动刷新',
    )
  } catch (e: any) {
    // 回滚 UI
    autoRefreshEnabled.value = !enabled
    ElMessage.error(e?.message || '保存失败')
  } finally {
    autoRefreshSaving.value = false
  }
}

// ========== 批量删除 ==========
const BULK_DELETE_LABELS: Record<string, string> = {
  dead:       '失效账号',
  suspicious: '可疑 / 已封账号',
  warned:     '风险账号',
  throttled:  '限流账号',
  all:        '全部账号',
}
async function onBulkDelete(scope: accountApi.BulkDeleteScope) {
  const label = BULK_DELETE_LABELS[scope] || scope
  try {
    await ElMessageBox.confirm(
      `确认将「${label}」全部删除?此操作会软删所有匹配条目,不可在当前界面恢复。`,
      scope === 'all' ? '⚠ 删除全部账号' : '批量删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: scope === 'all' ? 'error' : 'warning',
      },
    )
  } catch { return }
  try {
    const r = await accountApi.bulkDeleteAccounts(scope)
    ElMessage.success(`已删除 ${r.deleted} 个账号`)
    pager.page = 1
    fetchList()
  } catch (e: any) {
    ElMessage.error(e?.message || '删除失败')
  }
}

// ========== 日期工具(兼容 sql.NullTime 返回形态) ==========
function asDate(v: any): string {
  if (!v) return ''
  if (typeof v === 'string') return v
  if (typeof v === 'object') {
    if ('Valid' in v && !v.Valid) return ''
    if ('Time' in v) return v.Time
  }
  return ''
}
function fmtTime(v: any) {
  const s = asDate(v)
  return s ? formatDateShort(s) : '-'
}

// ========== 状态/类型展示 ==========
type TagType = 'success' | 'warning' | 'info' | 'danger' | 'primary'
const statusMap: Record<string, { label: string; type: TagType }> = {
  healthy:    { label: '健康',   type: 'success' },
  warned:     { label: '风险',   type: 'warning' },
  throttled:  { label: '限流',   type: 'warning' },
  suspicious: { label: '可疑',   type: 'info'    },
  dead:       { label: '失效',   type: 'danger'  },
}
function statusText(s: string): string { return statusMap[s]?.label || s || '-' }
function statusType(s: string): TagType { return statusMap[s]?.type || 'info' }

function typeLabel(t: string) {
  const map: Record<string, string> = { codex: 'Codex', chatgpt: 'ChatGPT', openai: 'OpenAI' }
  return map[t] || t || '-'
}

// ========== 即将过期高亮 ==========
function expiresClass(v: any): string {
  const s = asDate(v)
  if (!s) return 'muted'
  const t = new Date(s).getTime()
  if (Number.isNaN(t)) return 'muted'
  const diffMin = (t - Date.now()) / 60000
  if (diffMin < 0) return 'err'
  if (diffMin < 30) return 'warn'
  return ''
}

// ========== 新建 / 编辑 ==========
const dlg = ref(false)
const isEdit = ref(false)
const submitting = ref(false)
const formDefault = {
  id: 0,
  email: '',
  auth_token: '',
  refresh_token: '',
  session_token: '',
  token_expires_at: '',
  oai_session_id: '',
  oai_device_id: '',
  client_id: 'app_EMoamEEZ73f0CkXaXp7hrann',
  chatgpt_account_id: '',
  account_type: 'codex',
  plan_type: 'plus',
  daily_image_quota: 100,
  notes: '',
  cookies: '',
  proxy_id: 0,
  status: 'healthy',
}
const form = reactive({ ...formDefault })

function openCreate() {
  isEdit.value = false
  Object.assign(form, { ...formDefault })
  dlg.value = true
}

const secretsLoading = ref(false)

async function openEdit(row: accountApi.Account) {
  isEdit.value = true
  Object.assign(form, {
    id: row.id,
    email: row.email,
    auth_token: '',
    refresh_token: '',
    session_token: '',
    token_expires_at: asDate(row.token_expires_at),
    oai_session_id: row.oai_session_id || '',
    oai_device_id: row.oai_device_id || '',
    client_id: row.client_id || formDefault.client_id,
    chatgpt_account_id: row.chatgpt_account_id || '',
    account_type: row.account_type || 'codex',
    plan_type: row.plan_type || 'plus',
    daily_image_quota: row.daily_image_quota || 100,
    notes: row.notes || '',
    cookies: '',
    proxy_id: 0,
    status: row.status || 'healthy',
  })
  dlg.value = true
  // 异步拉取 AT / RT / ST 明文并回填,方便查看/修改
  secretsLoading.value = true
  try {
    const s = await accountApi.getAccountSecrets(row.id)
    form.auth_token    = s.auth_token    || ''
    form.refresh_token = s.refresh_token || ''
    form.session_token = s.session_token || ''
  } catch (e: any) {
    ElMessage.warning('未能加载 AT/RT/ST 明文,留空即不修改')
  } finally {
    secretsLoading.value = false
  }
}

async function copyText(text: string, label: string) {
  if (!text) { ElMessage.info('内容为空'); return }
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(`${label} 已复制`)
  } catch {
    ElMessage.error('复制失败,请手动选中复制')
  }
}

async function submitForm() {
  if (!form.email) { ElMessage.warning('请输入邮箱'); return }
  submitting.value = true
  try {
    if (!isEdit.value) {
      if (!form.auth_token) { ElMessage.warning('新建账号必须提供 access_token'); submitting.value = false; return }
      await accountApi.createAccount({ ...form })
      ElMessage.success('创建成功')
    } else {
      const body: any = {
        email: form.email,
        plan_type: form.plan_type,
        daily_image_quota: form.daily_image_quota,
        client_id: form.client_id,
        chatgpt_account_id: form.chatgpt_account_id,
        account_type: form.account_type,
        notes: form.notes,
        status: form.status,
      }
      if (form.auth_token)    body.auth_token    = form.auth_token
      if (form.refresh_token) body.refresh_token = form.refresh_token
      if (form.session_token) body.session_token = form.session_token
      if (form.cookies)       body.cookies       = form.cookies
      if (form.token_expires_at) body.token_expires_at = form.token_expires_at
      await accountApi.updateAccount(form.id, body)
      ElMessage.success('更新成功')
    }
    dlg.value = false
    await fetchList()
  } catch (e: any) {
    ElMessage.error(e?.message || '提交失败')
  } finally {
    submitting.value = false
  }
}

async function onDelete(row: accountApi.Account) {
  try {
    await ElMessageBox.confirm(`确定删除账号「${row.email}」?该操作不可恢复。`, '删除确认', {
      confirmButtonText: '删除', cancelButtonText: '取消', type: 'warning',
    })
  } catch { return }
  try {
    await accountApi.deleteAccount(row.id)
    ElMessage.success('已删除')
    fetchList()
  } catch (e: any) {
    ElMessage.error(e?.message || '删除失败')
  }
}

// ========== 绑定代理 ==========
const bindDlg = ref(false)
const bindForm = reactive({ id: 0, email: '', proxy_id: 0 })
function openBind(row: accountApi.Account) {
  bindForm.id = row.id
  bindForm.email = row.email
  bindForm.proxy_id = 0
  bindDlg.value = true
}
async function submitBind() {
  try {
    if (bindForm.proxy_id > 0) {
      await accountApi.bindProxy(bindForm.id, bindForm.proxy_id)
      ElMessage.success('已绑定代理')
    } else {
      await accountApi.unbindProxy(bindForm.id)
      ElMessage.success('已解绑')
    }
    bindDlg.value = false
    fetchList()
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  }
}

// ========== 刷新 / 探测(单条) ==========
const refreshingIds = ref<Set<number>>(new Set())
const probingIds = ref<Set<number>>(new Set())

async function onRefreshOne(row: accountApi.Account) {
  refreshingIds.value.add(row.id)
  try {
    const r = await accountApi.refreshAccount(row.id)
    if (r.ok) {
      if (r.at_verified === false) {
        // RT 刷成功但新 AT 没通过 chatgpt.com web 校验,提示用户
        ElMessage.warning(
          `刷新成功(来源:${r.source.toUpperCase()}),但新 AT 未通过 chatgpt.com 校验,可能无法用于聊天/图像接口`
        )
      } else {
        ElMessage.success(`刷新成功(来源:${r.source.toUpperCase()})`)
      }
    } else if (r.web_unauthorized) {
      ElMessage.error(
        r.error || 'RT 换出的 AT 被 chatgpt.com 拒绝,请为该账号补充 Session Token'
      )
    } else {
      ElMessage.error(r.error || '刷新失败')
    }
    fetchList()
  } catch (e: any) {
    ElMessage.error(e?.message || '刷新失败')
  } finally {
    refreshingIds.value.delete(row.id)
  }
}
async function onProbeOne(row: accountApi.Account) {
  probingIds.value.add(row.id)
  try {
    const r = await accountApi.probeAccountQuota(row.id)
    if (r.ok) {
      const parts: string[] = [`生图剩余 ${r.remaining}`]
      if (r.default_model) parts.push(`模型 ${r.default_model}`)
      if (r.blocked_features && r.blocked_features.length) {
        parts.push(`受限:${r.blocked_features.join(',')}`)
      }
      ElMessage.success(parts.join(' · '))
    } else {
      ElMessage.error(r.error || '探测失败')
    }
    fetchList()
  } catch (e: any) {
    ElMessage.error(e?.message || '探测失败')
  } finally {
    probingIds.value.delete(row.id)
  }
}

// ========== 全部刷新 / 全部探测 ==========
const batchRunning = ref<'none' | 'refresh' | 'probe'>('none')

async function onRefreshAll() {
  if (total.value === 0) { ElMessage.info('暂无账号'); return }
  try {
    await ElMessageBox.confirm(`将并发刷新全部账号(共 ${total.value} 个),可能耗时较久,是否继续?`, '批量刷新', {
      confirmButtonText: '开始', cancelButtonText: '取消',
    })
  } catch { return }
  batchRunning.value = 'refresh'
  try {
    const r = await accountApi.refreshAllAccounts()
    ElNotification.success({
      title: '批量刷新完成',
      message: `成功 ${r.success} · 失败 ${r.failed} · 合计 ${r.total}`,
      duration: 4000,
    })
    fetchList()
  } catch (e: any) {
    ElMessage.error(e?.message || '刷新失败')
  } finally {
    batchRunning.value = 'none'
  }
}

async function onProbeAll() {
  if (total.value === 0) { ElMessage.info('暂无账号'); return }
  try {
    await ElMessageBox.confirm(`将并发探测全部账号的图片额度(共 ${total.value} 个)?`, '批量探测', {
      confirmButtonText: '开始', cancelButtonText: '取消',
    })
  } catch { return }
  batchRunning.value = 'probe'
  try {
    const r = await accountApi.probeAllAccountsQuota()
    ElNotification.success({
      title: '批量探测完成',
      message: `成功 ${r.success} · 失败 ${r.failed} · 合计 ${r.total}`,
      duration: 4000,
    })
    fetchList()
  } catch (e: any) {
    ElMessage.error(e?.message || '探测失败')
  } finally {
    batchRunning.value = 'none'
  }
}

const importDlg = ref(false)
const importing = ref(false)
const importResultRows = ref<ImportDialogResultRow[]>([])

function openImport() {
  importResultRows.value = []
  importDlg.value = true
}

function clearImportResults() {
  importResultRows.value = []
}

function showImportSummary(summary: accountApi.ImportSummary, title = '账号导入完成') {
  const message = `新增 ${summary.created} · 更新 ${summary.updated} · 跳过 ${summary.skipped} · 失败 ${summary.failed}`
  if (summary.failed > 0 || summary.skipped > 0) {
    ElNotification.warning({ title, message, duration: 5000 })
    return
  }
  ElNotification.success({ title, message, duration: 4000 })
}

function toResultRows(summary: accountApi.ImportSummary): ImportDialogResultRow[] {
  return summary.results.map((row) => ({ ...row }))
}

async function submitJSONLikeImport(
  model: FileImportModel,
  sourceKind: 'cpa_file' | 'sub2api_json',
  payload: DialogSubmitPayload,
) {
  const options = {
    source_kind: sourceKind,
    update_existing: payload.advanced.update_existing,
    default_proxy_id: payload.advanced.default_proxy_id,
    target_pool_id: payload.advanced.target_pool_id,
    resolve_identity: payload.advanced.resolve_identity,
    kick_refresh: payload.advanced.kick_refresh,
    kick_quota_probe: payload.advanced.kick_quota_probe,
  }
  if (model.files.length > 0) {
    return accountApi.importAccountsFiles(model.files, options)
  }
  return accountApi.importAccountsJSON({
    text: model.text,
    ...options,
  })
}

function assertNeverPayload(value: never): never {
  throw new Error(`unsupported import payload: ${JSON.stringify(value)}`)
}

async function handleImportSubmit(payload: DialogSubmitPayload) {
  importing.value = true
  clearImportResults()
  try {
    switch (payload.kind) {
      case 'access_token': {
        const result = await accountApi.importAccountsTokens({
          mode: payload.payload.mode,
          tokens: payload.payload.tokens_text,
          client_id: payload.payload.client_id.trim() || undefined,
          update_existing: payload.advanced.update_existing,
          default_proxy_id: payload.advanced.default_proxy_id,
          target_pool_id: payload.advanced.target_pool_id,
          resolve_identity: payload.advanced.resolve_identity,
          kick_refresh: payload.advanced.kick_refresh,
          kick_quota_probe: payload.advanced.kick_quota_probe,
        })
        importResultRows.value = toResultRows(result)
        showImportSummary(result, `${payload.payload.mode.toUpperCase()} 导入完成`)
        break
      }
      case 'cpa':
      case 'sub2api': {
        const result = await submitJSONLikeImport(
          payload.payload,
          payload.kind === 'cpa' ? 'cpa_file' : 'sub2api_json',
          payload,
        )
        importResultRows.value = toResultRows(result)
        showImportSummary(result, payload.kind === 'cpa' ? 'CPA 导入完成' : 'sub2api 导入完成')
        break
      }
      case 'manual': {
        const body: accountApi.AccountCreate & { resolve_identity?: boolean } = {
          email: payload.payload.email.trim(),
          auth_token: payload.payload.auth_token.trim(),
          refresh_token: payload.payload.refresh_token.trim() || undefined,
          session_token: payload.payload.session_token.trim() || undefined,
          client_id: payload.payload.client_id.trim() || undefined,
          account_type: payload.payload.account_type,
          plan_type: payload.payload.plan_type,
          daily_image_quota: payload.payload.daily_image_quota,
          notes: payload.payload.notes,
          proxy_id: payload.advanced.default_proxy_id,
          target_pool_id: payload.advanced.target_pool_id,
          update_existing: payload.advanced.update_existing,
          resolve_identity: payload.advanced.resolve_identity,
          kick_refresh: payload.advanced.kick_refresh,
          kick_quota_probe: payload.advanced.kick_quota_probe,
        }
        const account = await accountApi.createAccount(body)
        importResultRows.value = [{
          index: 1,
          source_type: 'manual',
          source_ref: payload.payload.email.trim(),
          email: account.email,
          status: 'created',
          id: account.id,
        }]
        ElNotification.success({
          title: '手动新增完成',
          message: `已写入账号 ${account.email}`,
          duration: 4000,
        })
        break
      }
      default:
        assertNeverPayload(payload)
    }
    await fetchList()
  } catch (e: any) {
    ElMessage.error(e?.message || '导入失败')
  } finally {
    importing.value = false
  }
}

onMounted(() => {
  fetchList()
  fetchProxies()
  fetchPools()
  loadAutoRefresh()
})
</script>

<template>
  <div class="page-container">
    <!-- 顶栏:标题 + 动作 -->
    <div class="card-block hdr">
      <div class="flex-between">
        <div class="hdr-left">
          <h2 class="page-title">GPT 账号池</h2>
          <div class="page-sub">
            统一管理 ChatGPT Plus / Team / Codex 账号:JSON / AT / RT / ST 批量导入 · 自动刷新 · 图片额度探测 · 风控熔断轮转
          </div>
        </div>
        <div class="actions">
          <el-button :loading="batchRunning === 'probe'" :disabled="loading" @click="onProbeAll">
            全部探测
          </el-button>
          <el-button :loading="batchRunning === 'refresh'" :disabled="loading" @click="onRefreshAll">
            全部刷新
          </el-button>
          <el-dropdown trigger="click" @command="onBulkDelete">
            <el-button>批量删除</el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="dead">删除失效账号</el-dropdown-item>
                <el-dropdown-item command="suspicious">删除可疑/已封账号</el-dropdown-item>
                <el-dropdown-item command="warned">删除风险账号</el-dropdown-item>
                <el-dropdown-item command="throttled">删除限流账号</el-dropdown-item>
                <el-dropdown-item divided command="all">
                  <span style="color: var(--el-color-danger)">删除全部账号</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
          <el-button @click="openImport">批量导入</el-button>
          <el-button type="primary" @click="openCreate">新建账号</el-button>
        </div>
      </div>
    </div>

    <!-- 筛选栏 -->
    <div class="card-block">
      <el-form :inline="true" size="default" class="filter-form" @submit.prevent="onSearch">
        <el-form-item label="状态">
          <el-select v-model="filter.status" placeholder="全部" clearable style="width: 140px">
            <el-option label="全部" value="" />
            <el-option label="健康" value="healthy" />
            <el-option label="风险" value="warned" />
            <el-option label="限流" value="throttled" />
            <el-option label="可疑" value="suspicious" />
            <el-option label="失效" value="dead" />
          </el-select>
        </el-form-item>
        <el-form-item label="关键词">
          <el-input
            v-model="filter.keyword"
            placeholder="邮箱 / 备注"
            clearable
            style="width: 260px"
            @keyup.enter="onSearch"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="onSearch">搜索</el-button>
          <el-button @click="onReset">重置</el-button>
        </el-form-item>
        <el-form-item class="auto-refresh-item">
          <el-tooltip
            placement="top"
            content="开启后:AT 距离过期 < 1 天的账号会被后台自动续期;状态为「失效 / 可疑」的账号不会刷新"
          >
            <el-checkbox
              v-model="autoRefreshEnabled"
              :disabled="autoRefreshSaving"
              @change="onToggleAutoRefresh"
            >
              自动刷新 AT
              <span class="auto-refresh-hint">(&lt; 1 天过期时)</span>
            </el-checkbox>
          </el-tooltip>
        </el-form-item>
      </el-form>
    </div>

    <!-- 表格 -->
    <div class="card-block">
      <el-table
        v-loading="loading" :data="rows" stripe size="default" row-key="id"
        table-layout="auto" style="width: 100%"
      >
        <el-table-column label="邮箱" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <el-tooltip
              v-if="row.notes"
              placement="top"
              :content="row.notes"
            >
              <span class="email">{{ row.email }}</span>
            </el-tooltip>
            <span v-else class="email">{{ row.email }}</span>
          </template>
        </el-table-column>
        <el-table-column label="类型" width="76">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ typeLabel(row.account_type) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="76">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusText(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="凭证" width="86">
          <template #default="{ row }">
            <div class="creds">
              <el-tooltip content="存在 Refresh Token,可用 RT 自动刷新 AT" placement="top">
                <el-tag :type="row.has_rt ? 'success' : 'info'" size="small" effect="plain">
                  {{ row.has_rt ? 'RT' : '—' }}
                </el-tag>
              </el-tooltip>
              <el-tooltip content="存在 Session Token,可用 ST 回退刷新" placement="top">
                <el-tag :type="row.has_st ? 'success' : 'info'" size="small" effect="plain">
                  {{ row.has_st ? 'ST' : '—' }}
                </el-tag>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="AT 过期" min-width="148" show-overflow-tooltip>
          <template #default="{ row }">
            <span :class="expiresClass(row.token_expires_at)">{{ fmtTime(row.token_expires_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="生图剩余" width="96" align="center">
          <template #default="{ row }">
            <template v-if="row.image_quota_remaining >= 0">
              <el-tooltip
                placement="top"
                :disabled="!asDate(row.image_quota_reset_at)"
                :content="'下次重置:' + fmtTime(row.image_quota_reset_at)"
              >
                <span class="quota"><b>{{ row.image_quota_remaining }}</b></span>
              </el-tooltip>
            </template>
            <span v-else class="muted">未探测</span>
          </template>
        </el-table-column>
        <el-table-column label="今日已用 / 上限" width="140" align="center">
          <template #default="{ row }">
            <el-tooltip placement="top">
              <template #content>
                <div style="line-height:1.8">
                  <div>今日已用:{{ row.today_used_count }} 张</div>
                  <div>
                    账号真实额度:<b>
                      <template v-if="row.image_quota_total > 0">
                        {{ row.image_quota_total }}
                      </template>
                      <template v-else>未探测</template>
                    </b>
                    <span v-if="row.image_quota_remaining >= 0" style="color:#a1a5ad">
                      (剩余 {{ row.image_quota_remaining }})
                    </span>
                  </div>
                  <div>熔断上限(手工):{{ row.daily_image_quota }} / 日</div>
                </div>
              </template>
              <span class="quota">
                <b>{{ row.today_used_count }}</b>
                <span class="muted"> / </span>
                <template v-if="row.image_quota_total > 0">
                  <b>{{ row.image_quota_total }}</b>
                </template>
                <template v-else>
                  <span class="muted">{{ row.daily_image_quota }}</span>
                </template>
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="最近刷新" min-width="148" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="refresh-cell">
              <span>{{ fmtTime(row.last_refresh_at) }}</span>
              <el-tag
                v-if="row.last_refresh_source"
                size="small" effect="plain"
                :type="row.last_refresh_source === 'rt' ? 'success' : 'warning'"
              >{{ row.last_refresh_source.toUpperCase() }}</el-tag>
            </div>
            <el-tooltip
              v-if="row.refresh_error"
              placement="top"
              :content="row.refresh_error"
            >
              <div class="err">{{ row.refresh_error }}</div>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button
              link type="primary" size="small"
              :loading="refreshingIds.has(row.id)"
              @click="onRefreshOne(row)"
            >刷新</el-button>
            <el-button
              link type="primary" size="small"
              :loading="probingIds.has(row.id)"
              @click="onProbeOne(row)"
            >探测</el-button>
            <el-button link type="primary" size="small" @click="openBind(row)">代理</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger"  size="small" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="pager.page"
          v-model:page-size="pager.page_size"
          :total="total"
          :page-sizes="[10, 20, 50, 100, 200, 500, 1000]"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="fetchList"
          @size-change="fetchList"
        />
      </div>
    </div>

    <!-- 新建 / 编辑弹窗 -->
    <el-dialog v-model="dlg" :title="isEdit ? '编辑账号' : '新建账号'" width="720px" destroy-on-close>
      <el-form label-width="120px" size="default">
        <el-form-item label="邮箱">
          <el-input v-model="form.email" placeholder="user@example.com" />
        </el-form-item>
        <el-form-item label="账号类型">
          <el-select v-model="form.account_type" style="width: 180px">
            <el-option label="Codex" value="codex" />
            <el-option label="ChatGPT" value="chatgpt" />
          </el-select>
        </el-form-item>
        <el-form-item label="Access Token">
          <div class="token-field">
            <el-input
              v-model="form.auth_token"
              type="textarea" :rows="3"
              :placeholder="isEdit
                ? (secretsLoading ? '正在加载当前 AT……' : '当前为空,粘贴新的 access_token 可更新')
                : '粘贴 access_token(必填)'"
              spellcheck="false"
            />
            <el-button
              v-if="isEdit"
              size="small" link
              :disabled="!form.auth_token"
              @click="copyText(form.auth_token, 'Access Token')"
            >复制</el-button>
          </div>
        </el-form-item>
        <el-form-item label="Refresh Token">
          <div class="token-field">
            <el-input
              v-model="form.refresh_token"
              type="textarea" :rows="2"
              :placeholder="isEdit
                ? (secretsLoading ? '正在加载当前 RT……' : '该账号暂无 Refresh Token')
                : '可选;有 RT 则支持自动刷新'"
              spellcheck="false"
            />
            <el-button
              v-if="isEdit"
              size="small" link
              :disabled="!form.refresh_token"
              @click="copyText(form.refresh_token, 'Refresh Token')"
            >复制</el-button>
          </div>
        </el-form-item>
        <el-form-item label="Session Token">
          <div class="token-field">
            <el-input
              v-model="form.session_token"
              type="textarea" :rows="2"
              :placeholder="isEdit
                ? (secretsLoading ? '正在加载当前 ST……' : '该账号暂无 Session Token')
                : '可选;__Secure-next-auth.session-token 的值'"
              spellcheck="false"
            />
            <el-button
              v-if="isEdit"
              size="small" link
              :disabled="!form.session_token"
              @click="copyText(form.session_token, 'Session Token')"
            >复制</el-button>
          </div>
        </el-form-item>
        <el-form-item label="Token 过期时间">
          <el-date-picker
            v-model="form.token_expires_at"
            type="datetime" format="YYYY-MM-DD HH:mm:ss" value-format="YYYY-MM-DDTHH:mm:ssZ"
            placeholder="留空则从 JWT 自动解析"
            style="width: 260px"
          />
        </el-form-item>
        <el-form-item label="Client ID">
          <el-input v-model="form.client_id" />
        </el-form-item>
        <el-form-item label="ChatGPT AccountID">
          <el-input v-model="form.chatgpt_account_id" placeholder="可选;JSON 里有则自动填充" />
        </el-form-item>
        <el-form-item label="套餐">
          <el-select v-model="form.plan_type" style="width: 180px">
            <el-option label="Plus"  value="plus" />
            <el-option label="Team"  value="team" />
            <el-option label="Free"  value="free" />
            <el-option label="Codex" value="codex" />
          </el-select>
        </el-form-item>
        <el-form-item label="每日图片额度">
          <el-input-number v-model="form.daily_image_quota" :min="0" :max="10000" />
        </el-form-item>
        <el-form-item v-if="isEdit" label="状态">
          <el-select v-model="form.status" style="width: 180px">
            <el-option label="健康"  value="healthy" />
            <el-option label="风险"  value="warned" />
            <el-option label="限流"  value="throttled" />
            <el-option label="可疑"  value="suspicious" />
            <el-option label="失效"  value="dead" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.notes" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item v-if="!isEdit" label="绑定代理">
          <el-select v-model="form.proxy_id" clearable placeholder="不绑定" style="width: 100%">
            <el-option :value="0" label="不绑定" />
            <el-option
              v-for="p in proxies"
              :key="p.id"
              :label="`#${p.id} ${p.remark || p.host}:${p.port}`"
              :value="p.id"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dlg = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 绑定代理弹窗 -->
    <el-dialog v-model="bindDlg" title="绑定代理" width="420px">
      <div style="margin-bottom: 10px; color: var(--el-text-color-secondary)">
        账号:<b>{{ bindForm.email }}</b>
      </div>
      <el-select v-model="bindForm.proxy_id" clearable placeholder="选择代理(留空=解绑)" style="width: 100%">
        <el-option :value="0" label="不绑定 / 解绑" />
        <el-option
          v-for="p in proxies"
          :key="p.id"
          :label="`#${p.id} ${p.remark || p.host}:${p.port}`"
          :value="p.id"
        />
      </el-select>
      <template #footer>
        <el-button @click="bindDlg = false">取消</el-button>
        <el-button type="primary" @click="submitBind">确定</el-button>
      </template>
    </el-dialog>

    <AccountImportDialog
      v-model="importDlg"
      :loading="importing"
      :result-rows="importResultRows"
      :proxy-options="proxyOptions"
      :pool-options="poolOptions"
      @submit="handleImportSubmit"
    />
  </div>
</template>

<style scoped lang="scss">
.hdr { margin-bottom: 14px !important; }
.hdr-left .page-sub {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  margin-top: 4px;
}
.actions {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.flex-between {
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
}

.filter-form :deep(.el-form-item) { margin-bottom: 0; }
.auto-refresh-item { margin-left: 4px; }
.auto-refresh-hint {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  margin-left: 4px;
}

.email {
  color: var(--el-text-color-primary);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  display: inline-block; max-width: 100%;
}

.refresh-cell {
  display: flex; align-items: center; gap: 6px;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

.creds { display: flex; gap: 4px; }
.quota b { color: var(--el-color-primary); font-weight: 600; }
.muted { color: var(--el-text-color-secondary); }
.warn  { color: var(--el-color-warning); font-weight: 500; }
.err   {
  color: var(--el-color-danger);
  font-size: 12px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin-top: 2px;
}

.token-field {
  display: flex; align-items: flex-start; gap: 8px; width: 100%;
  :deep(.el-textarea) { flex: 1; }
}

.pager {
  display: flex; justify-content: flex-end;
  margin-top: 14px;
}

.file-list {
  margin-top: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  max-height: 200px;
  overflow: auto;
  padding: 8px 10px;
  .file-list-head {
    display: flex; justify-content: space-between; align-items: center;
    padding-bottom: 4px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    border-bottom: 1px dashed var(--el-border-color-lighter);
    margin-bottom: 4px;
  }
  .file-row {
    display: flex; align-items: center; justify-content: space-between;
    padding: 4px 0;
    font-size: 13px;
    .fname { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
    .fsize { color: var(--el-text-color-secondary); margin: 0 8px; font-variant-numeric: tabular-nums; }
  }
}

.progress {
  margin-top: 16px;
  padding: 12px 14px;
  background: var(--el-fill-color-lighter);
  border-radius: 8px;
  .progress-head {
    display: flex; justify-content: space-between; align-items: center;
    margin-bottom: 6px;
    .stat { display: flex; gap: 6px; }
  }
}

.err-list {
  margin-top: 12px;
  .err-list-head {
    color: var(--el-color-danger);
    font-weight: 500;
    margin-bottom: 6px;
  }
}
</style>

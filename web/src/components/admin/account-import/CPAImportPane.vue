<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { computed, onMounted, reactive, ref, watch } from 'vue'

import {
  createAccountImportSource,
  listAccountImportSources,
  listRemoteCPAFiles,
  type AccountImportSource,
  type CreateAccountImportSourceBody,
  type RemoteCPAFile,
} from '@/api/account-import-sources'

import type { FileImportModel } from './types'

const model = defineModel<FileImportModel>({
  required: true,
})

withDefaults(
  defineProps<{
    disabled?: boolean
  }>(),
  {
    disabled: false,
  },
)

const sources = ref<AccountImportSource[]>([])
const remoteFiles = ref<RemoteCPAFile[]>([])
const loadingSources = ref(false)
const loadingFiles = ref(false)
const createVisible = ref(false)
const savingSource = ref(false)

const createForm = reactive<CreateAccountImportSourceBody>({
  source_type: 'cpa',
  name: '',
  base_url: '',
  auth_mode: 'api_key',
  api_key: '',
  secret_key: '',
})

const cpaSources = computed(() => sources.value.filter((item) => item.source_type === 'cpa'))

watch(
  () => model.value.mode,
  (mode) => {
    if (mode === 'local') {
      model.value.source_id = undefined
      model.value.selected_remote_ids = []
      remoteFiles.value = []
    }
  },
)

function onPickFiles(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files) return
  model.value.files = Array.from(input.files)
  input.value = ''
}

function removeFile(index: number) {
  model.value.files.splice(index, 1)
}

async function fetchSources() {
  loadingSources.value = true
  try {
    const result = await listAccountImportSources()
    sources.value = result.items
  } catch {
    sources.value = []
  } finally {
    loadingSources.value = false
  }
}

async function fetchRemoteFiles() {
  if (!model.value.source_id) {
    ElMessage.warning('请先选择一个 CPA 远程源')
    return
  }
  loadingFiles.value = true
  try {
    const result = await listRemoteCPAFiles(model.value.source_id)
    remoteFiles.value = result.items
    model.value.selected_remote_ids = []
  } finally {
    loadingFiles.value = false
  }
}

async function createSource() {
  if (!createForm.name?.trim() || !createForm.base_url?.trim()) {
    ElMessage.warning('请填写名称和 base_url')
    return
  }
  if (createForm.auth_mode === 'api_key' && !createForm.api_key?.trim()) {
    ElMessage.warning('API Key 模式需要填写 api_key')
    return
  }
  if (createForm.auth_mode === 'bearer' && !createForm.secret_key?.trim()) {
    ElMessage.warning('Bearer 模式需要填写 secret_key')
    return
  }
  savingSource.value = true
  try {
    const created = await createAccountImportSource({
      source_type: 'cpa',
      name: createForm.name.trim(),
      base_url: createForm.base_url.trim(),
      auth_mode: createForm.auth_mode,
      api_key: createForm.auth_mode === 'api_key' ? createForm.api_key?.trim() : undefined,
      secret_key: createForm.auth_mode === 'bearer' ? createForm.secret_key?.trim() : undefined,
    })
    createVisible.value = false
    createForm.name = ''
    createForm.base_url = ''
    createForm.api_key = ''
    createForm.secret_key = ''
    await fetchSources()
    model.value.source_id = created.id
    ElMessage.success(`已创建远程源 ${created.name}`)
  } finally {
    savingSource.value = false
  }
}

function onSelectionChange(rows: RemoteCPAFile[]) {
  model.value.selected_remote_ids = rows.map((row) => row.name)
}

onMounted(() => {
  fetchSources()
})
</script>

<template>
  <div class="pane">
    <el-radio-group v-model="model.mode" :disabled="disabled" style="margin-bottom: 16px">
      <el-radio-button label="local">本地文件</el-radio-button>
      <el-radio-button label="remote">远程 CPA</el-radio-button>
    </el-radio-group>

    <el-form v-if="model.mode === 'local'" label-width="120px">
      <el-form-item label="CPA 文件">
        <div class="file-actions">
          <input accept=".json,.txt" :disabled="disabled" type="file" multiple @change="onPickFiles">
          <div class="pane-hint">支持直接上传 CPA JSON/TXT，或把内容粘贴到下方。</div>
        </div>
      </el-form-item>

      <el-form-item v-if="model.files.length" label="已选文件">
        <div class="file-list">
          <el-tag
            v-for="(file, index) in model.files"
            :key="`${file.name}-${index}`"
            closable
            :disable-transitions="true"
            @close="removeFile(index)"
          >
            {{ file.name }}
          </el-tag>
        </div>
      </el-form-item>

      <el-form-item label="文本内容">
        <el-input
          v-model="model.text"
          :autosize="{ minRows: 8, maxRows: 14 }"
          :disabled="disabled"
          type="textarea"
          placeholder="可直接粘贴 CPA JSON 内容"
        />
      </el-form-item>
    </el-form>

    <div v-else>
      <div class="remote-toolbar">
        <el-select
          v-model="model.source_id"
          :disabled="disabled || loadingSources"
          clearable
          filterable
          placeholder="选择一个远程 CPA 源"
          style="min-width: 280px"
        >
          <el-option
            v-for="item in cpaSources"
            :key="item.id"
            :label="item.name"
            :value="item.id"
          />
        </el-select>
        <el-button :disabled="disabled" :loading="loadingSources" @click="fetchSources">刷新源</el-button>
        <el-button :disabled="disabled" @click="createVisible = true">新增源</el-button>
        <el-button type="primary" :disabled="disabled || !model.source_id" :loading="loadingFiles" @click="fetchRemoteFiles">
          拉取文件
        </el-button>
      </div>

      <div class="pane-hint" style="margin-bottom: 12px">
        远程模式会从已保存的 CPA 源拉取文件列表，勾选后导入。
      </div>

      <el-table
        :data="remoteFiles"
        max-height="320"
        size="small"
        @selection-change="onSelectionChange"
      >
        <el-table-column type="selection" width="48" />
        <el-table-column prop="name" label="文件名" min-width="240" />
        <el-table-column prop="email" label="邮箱" min-width="200" />
      </el-table>
    </div>

    <el-dialog v-model="createVisible" title="新增 CPA 远程源" width="520px" append-to-body>
      <el-form label-width="110px">
        <el-form-item label="名称">
          <el-input v-model="createForm.name" placeholder="例如 CPA 主池" />
        </el-form-item>
        <el-form-item label="Base URL">
          <el-input v-model="createForm.base_url" placeholder="https://example.com" />
        </el-form-item>
        <el-form-item label="认证方式">
          <el-select v-model="createForm.auth_mode" style="width: 100%">
            <el-option label="X-API-Key" value="api_key" />
            <el-option label="Bearer" value="bearer" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="createForm.auth_mode === 'api_key'" label="API Key">
          <el-input v-model="createForm.api_key" show-password />
        </el-form-item>
        <el-form-item v-else label="Secret Key">
          <el-input v-model="createForm.secret_key" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingSource" @click="createSource">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.file-actions {
  width: 100%;
}

.file-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.pane-hint {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.remote-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
</style>
